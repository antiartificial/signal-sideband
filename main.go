package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"signal-sideband/pkg/ai"
	sig "signal-sideband/pkg/signal"
	"signal-sideband/pkg/store"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// 1. Setup Signal Client
	signalUrl := os.Getenv("SIGNAL_URL")
	if signalUrl == "" {
		signalUrl = "ws://localhost:8080/v1/receive"
	}
	client := sig.NewClient(signalUrl)

	// 2. Setup Store
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	// Construct generic connection string, or usage Supabase specific one if provided
	// For local docker/postgres: "postgres://user:pass@host:port/db"
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUser, dbPass, dbHost, dbPort, dbName)
	// Override if SUPABASE_URL/KEY is used? Usually Supabase uses standard postgres connection string (Session or Transaction pooler)
	// We'll trust the user has filled in DB_ details or put a full connection string in a variable.
	// For now, let's allow a direct DATABASE_URL override.
	if os.Getenv("DATABASE_URL") != "" {
		connStr = os.Getenv("DATABASE_URL")
	}

	ctx := context.Background()
	storage, err := store.NewStore(ctx, connStr)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v. Running in memory-only mode (logs only).", err)
	} else {
		log.Println("Connected to Database")
		defer storage.Close()
	}

	// 3. Setup AI
	var embedder ai.Embedder
	openAiKey := os.Getenv("OPENAI_API_KEY")
	if openAiKey != "" {
		log.Println("Using OpenAI for embeddings")
		embedder = ai.NewOpenAIEmbedder(openAiKey)
	} else {
		log.Println("Warning: OPENAI_API_KEY not set. Using Mock Embedder (zero vectors).")
		embedder = &ai.MockEmbedder{}
	}

	// 4. Connect Signal
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to Signal: %v", err)
	}
	defer client.Close()

	// 5. Start Reaper
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			if storage != nil {
				if err := storage.Reaper(ctx); err != nil {
					log.Printf("Reaper error: %v", err)
				}
			}
		}
	}()

	log.Println("Signal Sideband Relay Started. Press Ctrl+C to exit.")

	// 6. Main Loop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	msgChan := client.Messages()

	for {
		select {
		case <-stop:
			log.Println("Shutting down...")
			return
		case msg := <-msgChan:
			handleMessage(ctx, msg, storage, embedder)
		}
	}
}

func handleMessage(ctx context.Context, msg sig.SignalMessage, storage *store.Store, embedder ai.Embedder) {
	// Simple handler: Log -> Embed -> Store

	// Determine content
	var content string
	var expiresAt *time.Time
	var sender string
	var signalId string

	if msg.Envelope.DataMessage != nil {
		content = msg.Envelope.DataMessage.Message
		sender = msg.Envelope.SourceNumber
		// If SourceNumber is empty, check Source (uuid)
		if sender == "" {
			sender = msg.Envelope.Source
		}

		signalId = fmt.Sprintf("%d", msg.Envelope.DataMessage.Timestamp)

		if msg.Envelope.DataMessage.ExpiresInSeconds > 0 {
			t := time.Now().Add(time.Duration(msg.Envelope.DataMessage.ExpiresInSeconds) * time.Second)
			expiresAt = &t
		}
	} else if msg.Envelope.SyncMessage != nil && msg.Envelope.SyncMessage.SentMessage != nil {
		// Outgoing message (sync)
		content = msg.Envelope.SyncMessage.SentMessage.Message
		sender = "self" // or the account ID
		signalId = fmt.Sprintf("%d", msg.Envelope.SyncMessage.SentMessage.Timestamp)

		if msg.Envelope.SyncMessage.SentMessage.ExpiresInSeconds > 0 {
			t := time.Now().Add(time.Duration(msg.Envelope.SyncMessage.SentMessage.ExpiresInSeconds) * time.Second)
			expiresAt = &t
		}
	} else {
		return // Unknown type
	}

	if content == "" {
		return // Skip empty messages (e.g. strict attachments without text)
	}

	log.Printf("Received message from %s: %s", sender, content)

	// Embed
	embedding, err := embedder.Embed(content)
	if err != nil {
		log.Printf("Embedding error: %v", err)
		return
	}

	// Store
	if storage != nil {
		record := store.MessageRecord{
			SignalID:  signalId,
			SenderID:  sender,
			Content:   content,
			Embedding: embedding,
			ExpiresAt: expiresAt,
		}
		if err := storage.SaveMessage(ctx, record); err != nil {
			log.Printf("Storage error: %v", err)
		} else {
			log.Println("Message stored.")
		}
	}
}
