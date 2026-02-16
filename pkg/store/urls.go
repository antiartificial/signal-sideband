package store

import (
	"context"
	"fmt"
	"strings"
)

type URLFilter struct {
	Domain *string
	Search *string
	Sender *string
	Tag    *string
	Sort   string // "date" (default), "alpha"
}

func (s *Store) SaveURL(ctx context.Context, u URLRecord) (string, error) {
	query := `
		INSERT INTO urls (message_id, url, domain)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	err := s.pool.QueryRow(ctx, query, u.MessageID, u.URL, u.Domain).Scan(&id)
	return id, err
}

func (s *Store) ListURLs(ctx context.Context, limit, offset int, filter URLFilter) ([]URLRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}

	baseFrom := `
		FROM urls u
		LEFT JOIN messages m ON m.id = u.message_id
		LEFT JOIN contacts c ON c.source_uuid = m.source_uuid
	`

	var conditions []string
	var args []any
	argIdx := 1

	if filter.Domain != nil {
		conditions = append(conditions, fmt.Sprintf("u.domain = $%d", argIdx))
		args = append(args, *filter.Domain)
		argIdx++
	}
	if filter.Search != nil {
		pattern := "%" + strings.ToLower(*filter.Search) + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(COALESCE(u.title,'')) LIKE $%d OR LOWER(COALESCE(u.description,'')) LIKE $%d OR LOWER(COALESCE(u.summary,'')) LIKE $%d OR LOWER(u.url) LIKE $%d OR LOWER(u.domain) LIKE $%d)",
			argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, pattern)
		argIdx++
	}
	if filter.Sender != nil {
		conditions = append(conditions, fmt.Sprintf(
			"COALESCE(NULLIF(c.alias,''), c.profile_name, m.sender_id, '') = $%d", argIdx))
		args = append(args, *filter.Sender)
		argIdx++
	}
	if filter.Tag != nil {
		conditions = append(conditions, fmt.Sprintf("$%d = ANY(u.tags)", argIdx))
		args = append(args, *filter.Tag)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	var total int
	countQuery := "SELECT COUNT(*) " + baseFrom + where
	err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Order
	orderBy := "u.created_at DESC"
	if filter.Sort == "alpha" {
		orderBy = "COALESCE(NULLIF(u.title,''), u.url) ASC"
	}

	dataQuery := fmt.Sprintf(`
		SELECT u.id, u.message_id, u.url, u.domain, COALESCE(u.title,''), COALESCE(u.description,''), COALESCE(u.image_url,''),
			COALESCE(u.summary,''), u.tags, u.fetched, u.created_at,
			COALESCE(NULLIF(c.alias,''), c.profile_name, m.sender_id, '')
		%s%s ORDER BY %s LIMIT $%d OFFSET $%d`,
		baseFrom, where, orderBy, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var urls []URLRecord
	for rows.Next() {
		var u URLRecord
		if err := rows.Scan(
			&u.ID, &u.MessageID, &u.URL, &u.Domain, &u.Title, &u.Description,
			&u.ImageURL, &u.Summary, &u.Tags, &u.Fetched, &u.CreatedAt, &u.SharedBy,
		); err != nil {
			return nil, 0, err
		}
		urls = append(urls, u)
	}
	return urls, total, nil
}

func (s *Store) GetUnfetchedURLs(ctx context.Context) ([]URLRecord, error) {
	query := `
		SELECT u.id, u.message_id, u.url, u.domain, COALESCE(u.title,''), COALESCE(u.description,''), COALESCE(u.image_url,''),
			COALESCE(u.summary,''), u.tags, u.fetched, u.created_at,
			COALESCE(NULLIF(c.alias,''), c.profile_name, m.sender_id, '')
		FROM urls u
		LEFT JOIN messages m ON m.id = u.message_id
		LEFT JOIN contacts c ON c.source_uuid = m.source_uuid
		WHERE u.fetched = false
		ORDER BY u.created_at ASC
		LIMIT 100
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []URLRecord
	for rows.Next() {
		var u URLRecord
		if err := rows.Scan(
			&u.ID, &u.MessageID, &u.URL, &u.Domain, &u.Title, &u.Description,
			&u.ImageURL, &u.Summary, &u.Tags, &u.Fetched, &u.CreatedAt, &u.SharedBy,
		); err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}

func (s *Store) GetUnsummarizedURLs(ctx context.Context) ([]URLRecord, error) {
	query := `
		SELECT u.id, u.message_id, u.url, u.domain, COALESCE(u.title,''), COALESCE(u.description,''), COALESCE(u.image_url,''),
			COALESCE(u.summary,''), u.tags, u.fetched, u.created_at,
			COALESCE(NULLIF(c.alias,''), c.profile_name, m.sender_id, '')
		FROM urls u
		LEFT JOIN messages m ON m.id = u.message_id
		LEFT JOIN contacts c ON c.source_uuid = m.source_uuid
		WHERE u.fetched = true AND u.summary IS NULL
		ORDER BY u.created_at ASC
		LIMIT 20
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []URLRecord
	for rows.Next() {
		var u URLRecord
		if err := rows.Scan(
			&u.ID, &u.MessageID, &u.URL, &u.Domain, &u.Title, &u.Description,
			&u.ImageURL, &u.Summary, &u.Tags, &u.Fetched, &u.CreatedAt, &u.SharedBy,
		); err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}

func (s *Store) UpdateURLSummary(ctx context.Context, id, summary string, tags []string) error {
	query := `UPDATE urls SET summary = $2, tags = $3 WHERE id = $1`
	_, err := s.pool.Exec(ctx, query, id, summary, tags)
	return err
}

func (s *Store) MarkURLFetched(ctx context.Context, id, title, description, imageURL, summary string, tags []string) error {
	query := `
		UPDATE urls SET fetched = true, title = $2, description = $3, image_url = $4, summary = $5, tags = $6
		WHERE id = $1
	`
	_, err := s.pool.Exec(ctx, query, id, title, description, imageURL, summary, tags)
	return err
}
