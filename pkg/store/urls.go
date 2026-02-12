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
		SELECT id, message_id, url, domain, COALESCE(title,''), COALESCE(description,''), COALESCE(image_url,''), fetched, created_at
		FROM urls
	`

	var args []any
	if domain != nil {
		countQuery += " WHERE domain = $1"
		dataQuery += " WHERE domain = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		args = []any{*domain, limit, offset}
	} else {
		dataQuery += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
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
			&u.ImageURL, &u.Fetched, &u.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		urls = append(urls, u)
	}
	return urls, total, nil
}

func (s *Store) GetUnfetchedURLs(ctx context.Context) ([]URLRecord, error) {
	query := `
		SELECT id, message_id, url, domain, COALESCE(title,''), COALESCE(description,''), COALESCE(image_url,''), fetched, created_at
		FROM urls WHERE fetched = false
		ORDER BY created_at ASC
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
			&u.ImageURL, &u.Fetched, &u.CreatedAt,
		); err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, nil
}

func (s *Store) MarkURLFetched(ctx context.Context, id, title, description, imageURL string) error {
	query := `
		UPDATE urls SET fetched = true, title = $2, description = $3, image_url = $4
		WHERE id = $1
	`
	_, err := s.pool.Exec(ctx, query, id, title, description, imageURL)
	return err
}
