package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	pgvector "github.com/pgvector/pgvector-go"
)

func (s *Store) SaveMessage(ctx context.Context, msg MessageRecord) (string, error) {
	query := `
		INSERT INTO messages (signal_id, sender_id, content, embedding, expires_at,
			group_id, source_uuid, is_outgoing, view_once, has_attachments, raw_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (signal_id) DO NOTHING
		RETURNING id
	`
	// Wrap embedding for pgvector
	var vec *pgvector.Vector
	if len(msg.Embedding) > 0 {
		v := pgvector.NewVector(msg.Embedding)
		vec = &v
	}

	var id string
	err := s.pool.QueryRow(ctx, query,
		msg.SignalID, msg.SenderID, msg.Content, vec, msg.ExpiresAt,
		msg.GroupID, msg.SourceUUID, msg.IsOutgoing, msg.ViewOnce, msg.HasAttachments, msg.RawJSON,
	).Scan(&id)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", nil // duplicate, not an error
		}
		return "", err
	}
	return id, nil
}

func (s *Store) ListMessages(ctx context.Context, filter MessageFilter) ([]MessageRecord, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, "(expires_at IS NULL OR expires_at > now())")

	if filter.GroupID != nil {
		conditions = append(conditions, fmt.Sprintf("group_id = $%d", argIdx))
		args = append(args, *filter.GroupID)
		argIdx++
	}
	if filter.SenderID != nil {
		conditions = append(conditions, fmt.Sprintf("(sender_id = $%d OR source_uuid = $%d)", argIdx, argIdx))
		args = append(args, *filter.SenderID)
		argIdx++
	}
	if filter.After != nil {
		conditions = append(conditions, fmt.Sprintf("created_at > $%d", argIdx))
		args = append(args, *filter.After)
		argIdx++
	}
	if filter.Before != nil {
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", argIdx))
		args = append(args, *filter.Before)
		argIdx++
	}
	if filter.HasMedia != nil && *filter.HasMedia {
		conditions = append(conditions, "has_attachments = true")
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM messages WHERE %s", where)
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch rows
	query := fmt.Sprintf(`
		SELECT id, signal_id, sender_id, content, group_id, source_uuid,
			is_outgoing, view_once, has_attachments, created_at
		FROM messages
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []MessageRecord
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(
			&m.ID, &m.SignalID, &m.SenderID, &m.Content, &m.GroupID, &m.SourceUUID,
			&m.IsOutgoing, &m.ViewOnce, &m.HasAttachments, &m.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		messages = append(messages, m)
	}
	return messages, total, nil
}

func (s *Store) SemanticSearch(ctx context.Context, embedding []float32, threshold float64, limit int) ([]SearchResult, error) {
	query := `
		SELECT id, signal_id, sender_id, content, group_id, source_uuid,
			is_outgoing, has_attachments, similarity, created_at
		FROM match_messages($1, $2, $3)
	`
	vec := pgvector.NewVector(embedding)
	rows, err := s.pool.Query(ctx, query, vec, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var sim float64
		if err := rows.Scan(
			&r.ID, &r.SignalID, &r.SenderID, &r.Content, &r.GroupID, &r.SourceUUID,
			&r.IsOutgoing, &r.HasAttachments, &sim, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		r.Similarity = &sim
		results = append(results, r)
	}
	return results, nil
}

func (s *Store) FullTextSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return s.FilteredFullTextSearch(ctx, query, SearchFilter{}, limit)
}

func (s *Store) FilteredFullTextSearch(ctx context.Context, query string, filter SearchFilter, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}

	var conditions []string
	var args []any
	argIdx := 1

	// Full text match
	conditions = append(conditions, fmt.Sprintf("m.tsv @@ plainto_tsquery('english', $%d)", argIdx))
	args = append(args, query)
	argIdx++

	conditions = append(conditions, "(m.expires_at IS NULL OR m.expires_at > now())")

	if filter.GroupID != nil {
		conditions = append(conditions, fmt.Sprintf("m.group_id = $%d", argIdx))
		args = append(args, *filter.GroupID)
		argIdx++
	}
	if filter.SenderID != nil {
		conditions = append(conditions, fmt.Sprintf("(m.sender_id = $%d OR m.source_uuid = $%d)", argIdx, argIdx))
		args = append(args, *filter.SenderID)
		argIdx++
	}
	if filter.After != nil {
		conditions = append(conditions, fmt.Sprintf("m.created_at > $%d", argIdx))
		args = append(args, *filter.After)
		argIdx++
	}
	if filter.Before != nil {
		conditions = append(conditions, fmt.Sprintf("m.created_at < $%d", argIdx))
		args = append(args, *filter.Before)
		argIdx++
	}
	if filter.HasMedia != nil && *filter.HasMedia {
		conditions = append(conditions, "m.has_attachments = true")
	}

	where := strings.Join(conditions, " AND ")

	sqlQuery := fmt.Sprintf(`
		SELECT m.id, m.signal_id, m.sender_id, m.content, m.group_id, m.source_uuid,
			m.is_outgoing, m.has_attachments,
			ts_rank(m.tsv, plainto_tsquery('english', $1)) AS rank,
			m.created_at
		FROM messages m
		WHERE %s
		ORDER BY rank DESC
		LIMIT $%d
	`, where, argIdx)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var rank float32
		if err := rows.Scan(
			&r.ID, &r.SignalID, &r.SenderID, &r.Content, &r.GroupID, &r.SourceUUID,
			&r.IsOutgoing, &r.HasAttachments, &rank, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		r.Rank = &rank
		results = append(results, r)
	}
	return results, nil
}

func (s *Store) FilteredSemanticSearch(ctx context.Context, embedding []float32, threshold float64, filter SearchFilter, limit int) ([]SearchResult, error) {
	// Over-fetch then post-filter
	fetchLimit := limit * 5
	if fetchLimit > 200 {
		fetchLimit = 200
	}

	results, err := s.SemanticSearch(ctx, embedding, threshold, fetchLimit)
	if err != nil {
		return nil, err
	}

	var filtered []SearchResult
	for _, r := range results {
		if filter.GroupID != nil && (r.GroupID == nil || *r.GroupID != *filter.GroupID) {
			continue
		}
		if filter.SenderID != nil && r.SenderID != *filter.SenderID && (r.SourceUUID == nil || *r.SourceUUID != *filter.SenderID) {
			continue
		}
		if filter.After != nil && !r.CreatedAt.After(*filter.After) {
			continue
		}
		if filter.Before != nil && !r.CreatedAt.Before(*filter.Before) {
			continue
		}
		if filter.HasMedia != nil && *filter.HasMedia && !r.HasAttachments {
			continue
		}
		filtered = append(filtered, r)
		if len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

func (s *Store) PurgeMessagesNotInGroup(ctx context.Context, groupID string) (int, []string, error) {
	// Collect file paths before deleting
	pathQuery := `
		SELECT DISTINCT a.local_path FROM attachments a
		JOIN messages m ON a.message_id = m.id
		WHERE (m.group_id IS NULL OR m.group_id != $1)
		AND a.local_path != ''
		UNION
		SELECT DISTINCT a.thumbnail_path FROM attachments a
		JOIN messages m ON a.message_id = m.id
		WHERE (m.group_id IS NULL OR m.group_id != $1)
		AND a.thumbnail_path != ''
	`
	rows, err := s.pool.Query(ctx, pathQuery, groupID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return 0, nil, err
		}
		paths = append(paths, p)
	}

	// Delete messages (CASCADE handles attachments and urls)
	delQuery := `DELETE FROM messages WHERE group_id IS NULL OR group_id != $1`
	res, err := s.pool.Exec(ctx, delQuery, groupID)
	if err != nil {
		return 0, nil, err
	}

	return int(res.RowsAffected()), paths, nil
}

func (s *Store) GetMessagesByTimeRange(ctx context.Context, start, end time.Time, groupID *string) ([]MessageRecord, error) {
	var query string
	var args []any

	if groupID != nil {
		query = `
			SELECT id, signal_id, sender_id, content, group_id, source_uuid,
				is_outgoing, has_attachments, created_at
			FROM messages
			WHERE created_at >= $1 AND created_at <= $2 AND group_id = $3
			AND (expires_at IS NULL OR expires_at > now())
			ORDER BY created_at ASC
		`
		args = []any{start, end, *groupID}
	} else {
		query = `
			SELECT id, signal_id, sender_id, content, group_id, source_uuid,
				is_outgoing, has_attachments, created_at
			FROM messages
			WHERE created_at >= $1 AND created_at <= $2
			AND (expires_at IS NULL OR expires_at > now())
			ORDER BY created_at ASC
		`
		args = []any{start, end}
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageRecord
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(
			&m.ID, &m.SignalID, &m.SenderID, &m.Content, &m.GroupID, &m.SourceUUID,
			&m.IsOutgoing, &m.HasAttachments, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
