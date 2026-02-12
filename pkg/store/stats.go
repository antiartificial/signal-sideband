package store

import (
	"context"
)

func (s *Store) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{}

	// Total messages
	err := s.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM messages WHERE expires_at IS NULL OR expires_at > now()").
		Scan(&stats.TotalMessages)
	if err != nil {
		return nil, err
	}

	// Today's messages
	err = s.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM messages WHERE created_at >= CURRENT_DATE AND (expires_at IS NULL OR expires_at > now())").
		Scan(&stats.TodayMessages)
	if err != nil {
		return nil, err
	}

	// Total groups
	err = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM groups").Scan(&stats.TotalGroups)
	if err != nil {
		return nil, err
	}

	// Total URLs
	err = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM urls").Scan(&stats.TotalURLs)
	if err != nil {
		return nil, err
	}

	// Latest digest
	var d DigestRecord
	err = s.pool.QueryRow(ctx, `
		SELECT id, title, summary, topics, decisions, action_items,
			period_start, period_end, group_id, llm_provider, llm_model, token_count, created_at
		FROM digests ORDER BY created_at DESC LIMIT 1
	`).Scan(
		&d.ID, &d.Title, &d.Summary, &d.Topics, &d.Decisions, &d.ActionItems,
		&d.PeriodStart, &d.PeriodEnd, &d.GroupID, &d.LLMProvider, &d.LLMModel,
		&d.TokenCount, &d.CreatedAt,
	)
	if err == nil {
		stats.LatestDigest = &d
	}

	return stats, nil
}
