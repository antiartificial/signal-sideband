package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, connString string) (*Store, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) SearchSimilar(ctx context.Context, embedding []float32, threshold float64, limit int) ([]string, error) {
	query := `SELECT content FROM match_messages($1, $2, $3)`
	vec := pgvector.NewVector(embedding)
	rows, err := s.pool.Query(ctx, query, vec, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		results = append(results, content)
	}
	return results, nil
}

func (s *Store) Reaper(ctx context.Context) ([]string, error) {
	// Collect media paths from expired messages before deleting
	pathQuery := `
		SELECT a.local_path FROM attachments a
		JOIN messages m ON a.message_id = m.id
		WHERE m.expires_at < NOW() AND a.local_path != ''
		UNION
		SELECT a.thumbnail_path FROM attachments a
		JOIN messages m ON a.message_id = m.id
		WHERE m.expires_at < NOW() AND a.thumbnail_path != ''
	`
	rows, err := s.pool.Query(ctx, pathQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}

	query := `DELETE FROM messages WHERE expires_at < NOW()`
	res, err := s.pool.Exec(ctx, query)
	if err != nil {
		return nil, err
	}
	if n := res.RowsAffected(); n > 0 {
		fmt.Printf("Reaper: Deleted %d expired messages\n", n)
	}
	return paths, nil
}

func (s *Store) Close() {
	s.pool.Close()
}
