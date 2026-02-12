package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func setupTestStore(t *testing.T) *Store {
	_ = godotenv.Load("../../.env") // Try to load from root .env

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		t.Skip("Skipping integration test: DB_HOST not set")
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Fallback to constructing it
		dbUser := os.Getenv("DB_USER")
		dbPass := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		connStr = "postgres://" + dbUser + ":" + dbPass + "@" + dbHost + ":" + dbPort + "/" + dbName
	}

	ctx := context.Background()
	store, err := NewStore(ctx, connStr)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	return store
}

func TestSaveAndSearch(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()
	ctx := context.Background()

	// Clean up previous run
	_, _ = s.pool.Exec(ctx, "DELETE FROM messages WHERE signal_id LIKE 'test-%'")

	// 1. Save
	embedding := make([]float32, 1536)
	embedding[0] = 1.0 // Make it non-zero

	msg := MessageRecord{
		SignalID:  "test-1",
		SenderID:  "+15550001",
		Content:   "Hello World Test",
		Embedding: embedding,
	}

	_, err := s.SaveMessage(ctx, msg)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	// 2. Search
	// Exact match search
	results, err := s.SearchSimilar(ctx, embedding, 0.9, 1)
	if err != nil {
		t.Fatalf("SearchSimilar failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("Expected 1 result, got 0")
	}
	if results[0] != "Hello World Test" {
		t.Errorf("Expected content 'Hello World Test', got '%s'", results[0])
	}
}

func TestReaper(t *testing.T) {
	s := setupTestStore(t)
	defer s.Close()
	ctx := context.Background()

	// 1. Save Expired Message
	expiration := time.Now().Add(-1 * time.Hour) // 1 hour ago
	msg := MessageRecord{
		SignalID:  "test-expired-1",
		SenderID:  "+15550001",
		Content:   "This should be deleted",
		Embedding: make([]float32, 1536),
		ExpiresAt: &expiration,
	}

	if _, err := s.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("Failed to save expired message: %v", err)
	}

	// 2. Run Reaper
	if err := s.Reaper(ctx); err != nil {
		t.Fatalf("Reaper failed: %v", err)
	}

	// 3. Verify Deletion
	var count int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM messages WHERE signal_id = 'test-expired-1'").Scan(&count)
	if err != nil {
		t.Fatalf("Count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected expired message to be deleted, found %d", count)
	}
}
