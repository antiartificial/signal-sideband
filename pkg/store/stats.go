package store

import (
	"context"
	"encoding/json"
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

	// Latest daily insight (includes cached superlatives)
	insight, err := s.GetLatestInsight(ctx)
	if err == nil && insight != nil {
		stats.DailyInsight = insight

		// Use cached superlatives from insight if available
		if len(insight.Superlatives) > 2 { // more than "[]"
			var cached []Superlative
			if json.Unmarshal(insight.Superlatives, &cached) == nil && len(cached) > 0 {
				stats.Superlatives = cached
			}
		}
	}

	// Fallback: compute live if no cached superlatives
	if len(stats.Superlatives) == 0 {
		stats.Superlatives = s.GetSuperlatives(ctx)
	}

	return stats, nil
}

func (s *Store) GetLatestInsight(ctx context.Context) (*DailyInsight, error) {
	var di DailyInsight
	err := s.pool.QueryRow(ctx, `
		SELECT id, overview, themes, COALESCE(quote_content,''), COALESCE(quote_sender,''),
			quote_created_at, COALESCE(image_path,''), COALESCE(superlatives, '[]'::jsonb), created_at
		FROM daily_insights ORDER BY created_at DESC LIMIT 1
	`).Scan(
		&di.ID, &di.Overview, &di.Themes, &di.QuoteContent, &di.QuoteSender,
		&di.QuoteCreatedAt, &di.ImagePath, &di.Superlatives, &di.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &di, nil
}

func (s *Store) SaveDailyInsight(ctx context.Context, overview string, themes json.RawMessage, quoteContent, quoteSender string, superlatives json.RawMessage) (string, error) {
	query := `
		INSERT INTO daily_insights (overview, themes, quote_content, quote_sender, superlatives)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id string
	err := s.pool.QueryRow(ctx, query, overview, themes, quoteContent, quoteSender, superlatives).Scan(&id)
	return id, err
}

func (s *Store) SetInsightImagePath(ctx context.Context, id, imagePath string) error {
	_, err := s.pool.Exec(ctx, `UPDATE daily_insights SET image_path = $2 WHERE id = $1`, id, imagePath)
	return err
}

func (s *Store) GetRandomQuote(ctx context.Context) (string, string, error) {
	var content, sender string
	err := s.pool.QueryRow(ctx, `
		SELECT content, sender_id FROM messages
		WHERE content != '' AND LENGTH(content) > 20
		AND (expires_at IS NULL OR expires_at > now())
		ORDER BY random() LIMIT 1
	`).Scan(&content, &sender)
	return content, sender, err
}
