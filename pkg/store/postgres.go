package store

import (
	"context"
	"fmt"
	"time"

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

type MessageRecord struct {
	SignalID  string     `db:"signal_id"`
	SenderID  string     `db:"sender_id"`
	Content   string     `db:"content"`
	Embedding []float32  `db:"embedding"`
	ExpiresAt *time.Time `db:"expires_at"`
}

func (s *Store) SaveMessage(ctx context.Context, msg MessageRecord) error {
	query := `
		INSERT INTO messages (signal_id, sender_id, content, embedding, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (signal_id) DO NOTHING
	`
	// Use pgvectorcompatible array for embedding
	// pgx/v5 handles []float32 mapping to vector automatically if registered?
	// Actually we might need pgxvector library, but often standard array works if cast or if pgx defaults are good.
	// For simplicity, we assume generic array handling works or we might need `github.com/pgvector/pgvector-go`
	// But let's try standard slice.

	_, err := s.pool.Exec(ctx, query, msg.SignalID, msg.SenderID, msg.Content, msg.Embedding, msg.ExpiresAt)
	return err
}

func (s *Store) SearchSimilar(ctx context.Context, embedding []float32, threshold float64, limit int) ([]string, error) {
	query := `
		SELECT content 
		FROM match_messages($1, $2, $3)
	`
	rows, err := s.pool.Query(ctx, query, embedding, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var content string
		var sim float64
		var createdAt time.Time
		if err := rows.Scan(&content, &sim, &createdAt); err != nil {
			return nil, err
		}
		results = append(results, content)
	}
	return results, nil
}

// Reaper deletes expired messages
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
