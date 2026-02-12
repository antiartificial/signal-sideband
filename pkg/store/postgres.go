package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	rows, err := s.pool.Query(ctx, query, embedding, threshold, limit)
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

func (s *Store) Reaper(ctx context.Context) error {
	query := `DELETE FROM messages WHERE expires_at < NOW()`
	res, err := s.pool.Exec(ctx, query)
	if err != nil {
		return err
	}
	if rows := res.RowsAffected(); rows > 0 {
		fmt.Printf("Reaper: Deleted %d expired messages\n", rows)
	}
	return nil
}

func (s *Store) Close() {
	s.pool.Close()
}
