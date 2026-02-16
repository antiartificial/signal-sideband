package store

import (
	"context"
)

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

func (s *Store) ListURLs(ctx context.Context, limit, offset int, domain *string) ([]URLRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}

	countQuery := "SELECT COUNT(*) FROM urls"
	dataQuery := `
		SELECT u.id, u.message_id, u.url, u.domain, COALESCE(u.title,''), COALESCE(u.description,''), COALESCE(u.image_url,''),
			COALESCE(u.summary,''), u.tags, u.fetched, u.created_at,
			COALESCE(NULLIF(c.alias,''), c.profile_name, m.sender_id, '')
		FROM urls u
		LEFT JOIN messages m ON m.id = u.message_id
		LEFT JOIN contacts c ON c.source_uuid = m.source_uuid
	`

	var args []any
	if domain != nil {
		countQuery += " WHERE domain = $1"
		dataQuery += " WHERE u.domain = $1 ORDER BY u.created_at DESC LIMIT $2 OFFSET $3"
		args = []any{*domain, limit, offset}
	} else {
		dataQuery += " ORDER BY u.created_at DESC LIMIT $1 OFFSET $2"
		args = []any{limit, offset}
	}

	var total int
	if domain != nil {
		err := s.pool.QueryRow(ctx, countQuery, *domain).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	} else {
		err := s.pool.QueryRow(ctx, countQuery).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	}

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
