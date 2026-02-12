package store

import (
	"context"
)

func (s *Store) SaveDigest(ctx context.Context, d DigestRecord) (string, error) {
	query := `
		INSERT INTO digests (title, summary, topics, decisions, action_items,
			period_start, period_end, group_id, llm_provider, llm_model, token_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`
	var id string
	err := s.pool.QueryRow(ctx, query,
		d.Title, d.Summary, d.Topics, d.Decisions, d.ActionItems,
		d.PeriodStart, d.PeriodEnd, d.GroupID, d.LLMProvider, d.LLMModel, d.TokenCount,
	).Scan(&id)
	return id, err
}

func (s *Store) ListDigests(ctx context.Context, limit, offset int) ([]DigestRecord, int, error) {
	if limit <= 0 {
		limit = 20
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM digests").Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, title, summary, topics, decisions, action_items,
			period_start, period_end, group_id, llm_provider, llm_model, token_count, created_at
		FROM digests
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var digests []DigestRecord
	for rows.Next() {
		var d DigestRecord
		if err := rows.Scan(
			&d.ID, &d.Title, &d.Summary, &d.Topics, &d.Decisions, &d.ActionItems,
			&d.PeriodStart, &d.PeriodEnd, &d.GroupID, &d.LLMProvider, &d.LLMModel,
			&d.TokenCount, &d.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		digests = append(digests, d)
	}
	return digests, total, nil
}

func (s *Store) GetDigest(ctx context.Context, id string) (*DigestRecord, error) {
	query := `
		SELECT id, title, summary, topics, decisions, action_items,
			period_start, period_end, group_id, llm_provider, llm_model, token_count, created_at
		FROM digests WHERE id = $1
	`
	var d DigestRecord
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.Title, &d.Summary, &d.Topics, &d.Decisions, &d.ActionItems,
		&d.PeriodStart, &d.PeriodEnd, &d.GroupID, &d.LLMProvider, &d.LLMModel,
		&d.TokenCount, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
