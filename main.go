package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"signal-sideband/pkg/ai"
	"signal-sideband/pkg/api"
	"signal-sideband/pkg/digest"
	"signal-sideband/pkg/extract"
	"signal-sideband/pkg/llm"
	"signal-sideband/pkg/media"
	sig "signal-sideband/pkg/signal"
	"signal-sideband/pkg/store"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup Signal WebSocket Client
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

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUser, dbPass, dbHost, dbPort, dbName)
	if os.Getenv("DATABASE_URL") != "" {
		connStr = os.Getenv("DATABASE_URL")
	}

	storage, err := store.NewStore(ctx, connStr)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v. Running in memory-only mode.", err)
	} else {
		log.Println("Connected to database")
		defer storage.Close()
	}

	// 3. Setup Embedder
	var embedder ai.Embedder
	openAiKey := os.Getenv("OPENAI_API_KEY")
	if openAiKey != "" {
		log.Println("Using OpenAI for embeddings")
		embedder = ai.NewOpenAIEmbedder(openAiKey)
	} else {
		log.Println("Warning: OPENAI_API_KEY not set. Using mock embedder.")
		embedder = &ai.MockEmbedder{}
	}

	// 4. Setup Signal REST API client
	signalAPIURL := os.Getenv("SIGNAL_API_URL")
	if signalAPIURL == "" {
		signalAPIURL = "http://localhost:8080"
	}
	signalNumber := os.Getenv("SIGNAL_NUMBER")
	if signalNumber == "" {
		signalNumber = "+16619930050"
	}
	signalAPI := sig.NewAPIClient(signalAPIURL, signalNumber)

	// 5. Setup LLM provider (optional)
	var llmProvider llm.Provider
	var digestGen *digest.Generator
	llmProviderName := os.Getenv("LLM_PROVIDER")
	if llmProviderName != "" {
		p, err := llm.NewProvider(llmProviderName)
		if err != nil {
			log.Printf("Warning: LLM provider setup failed: %v", err)
		} else {
			llmProvider = p
			log.Printf("LLM provider: %s", llmProvider.Name())
		}
	}
	if llmProvider != nil && storage != nil {
		digestGen = digest.NewGenerator(storage, llmProvider)
	}

	// 6. Setup media path
	mediaPath := os.Getenv("MEDIA_PATH")
	if mediaPath == "" {
		mediaPath = "./media"
	}
	os.MkdirAll(mediaPath, 0755)

	// 7. Start API server
	authPassword := os.Getenv("AUTH_PASSWORD")

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "3001"
	}

	// Web directory for static frontend
	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		// Default: check for web/dist next to the binary
		if _, err := os.Stat("./web/dist/index.html"); err == nil {
			webDir = "./web/dist"
		}
	}

	if storage != nil {
		apiServer := api.NewServer(storage, embedder, digestGen, apiPort, authPassword, mediaPath, webDir)
		go func() {
			if err := apiServer.Start(); err != nil {
				log.Printf("API server error: %v", err)
			}
		}()
		defer apiServer.Shutdown(ctx)
	}

	// 8. Start background workers
	if storage != nil {
		// Reaper
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := storage.Reaper(ctx); err != nil {
						log.Printf("Reaper error: %v", err)
					}
				}
			}
		}()

		// Media download worker
		downloader := media.NewDownloader(signalAPI, mediaPath)
		mediaWorker := media.NewWorker(storage, downloader, 30*time.Second, mediaPath)
		go mediaWorker.Start(ctx)

		// AI vision analysis worker (requires XAI_API_KEY)
		if xaiKey := os.Getenv("XAI_API_KEY"); xaiKey != "" {
			analyzeWorker := media.NewAnalyzeWorker(storage, xaiKey, 60*time.Second, mediaPath)
			go analyzeWorker.Start(ctx)
		}

		// Link preview worker
		previewWorker := extract.NewPreviewWorker(storage, 60*time.Second)
		go previewWorker.Start(ctx)

		// Digest scheduler (daily at midnight)
		if digestGen != nil {
			insightsGen := digest.NewInsightsGenerator(storage, llmProvider)
			scheduler := digest.NewScheduler(digestGen, insightsGen, 24*time.Hour)
			go scheduler.Start(ctx)
		}
	}

	// 9. Initial group sync
	if storage != nil {
		go func() {
			syncGroups(ctx, signalAPI, storage)
		}()
	}

	// 10. Connect Signal WebSocket (non-fatal: retries in background)
	go func() {
		for {
			if err := client.Connect(); err != nil {
				log.Printf("Signal connection failed: %v (retrying in 30s)", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
					continue
				}
			}
			log.Println("Signal connected")
			// Read messages until disconnected
			for msg := range client.Messages() {
				handleMessage(ctx, msg, storage, embedder, signalAPI)
			}
			// If we get here, the channel closed — reconnect
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}()
	defer client.Close()

	log.Println("Signal Sideband started. Press Ctrl+C to exit.")

	// 11. Wait for shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down...")
	cancel()
}

func syncGroups(ctx context.Context, api *sig.APIClient, storage *store.Store) {
	groups, err := api.ListGroups()
	if err != nil {
		log.Printf("Group sync failed: %v", err)
		return
	}
	for _, g := range groups {
		if err := storage.UpsertGroup(ctx, store.GroupRecord{
			GroupID:     g.ID,
			Name:        g.Name,
			Description: g.Description,
			MemberCount: len(g.Members),
		}); err != nil {
			log.Printf("Group sync upsert failed for %s: %v", g.Name, err)
		}
	}
	log.Printf("Synced %d groups", len(groups))
}

func handleMessage(ctx context.Context, msg sig.SignalMessage, storage *store.Store, embedder ai.Embedder, signalAPI *sig.APIClient) {
	var content string
	var expiresAt *time.Time
	var sender string
	var signalId string
	var groupID *string
	var sourceUUID *string
	var isOutgoing bool
	var viewOnce bool
	var hasAttachments bool
	var attachments []sig.Attachment
	var dataMsg *sig.DataMessage

	if msg.Envelope.DataMessage != nil {
		dataMsg = msg.Envelope.DataMessage
		content = dataMsg.Message
		sender = msg.Envelope.SourceNumber
		if sender == "" {
			sender = msg.Envelope.Source
		}
		signalId = fmt.Sprintf("%d", dataMsg.Timestamp)
		isOutgoing = false

		if msg.Envelope.SourceUuid != "" {
			uuid := msg.Envelope.SourceUuid
			sourceUUID = &uuid
		}
	} else if msg.Envelope.SyncMessage != nil && msg.Envelope.SyncMessage.SentMessage != nil {
		dataMsg = msg.Envelope.SyncMessage.SentMessage
		content = dataMsg.Message
		sender = "self"
		signalId = fmt.Sprintf("%d", dataMsg.Timestamp)
		isOutgoing = true
	} else {
		return
	}

	if dataMsg.ExpiresInSeconds > 0 {
		t := time.Now().Add(time.Duration(dataMsg.ExpiresInSeconds) * time.Second)
		expiresAt = &t
	}

	viewOnce = dataMsg.ViewOnce

	if dataMsg.GroupInfo != nil {
		gid := dataMsg.GroupInfo.GroupId
		groupID = &gid
	}

	if len(dataMsg.Attachments) > 0 {
		hasAttachments = true
		attachments = dataMsg.Attachments
	}

	// Store raw JSON
	rawJSON, _ := json.Marshal(msg)

	if content == "" && !hasAttachments {
		return
	}

	log.Printf("Message from %s: %s", sender, truncate(content, 80))

	// Embed (if there's text content)
	var embedding []float32
	if content != "" {
		var err error
		embedding, err = embedder.Embed(content)
		if err != nil {
			log.Printf("Embedding error: %v", err)
		}
	}

	if storage == nil {
		return
	}

	// Save message
	record := store.MessageRecord{
		SignalID:       signalId,
		SenderID:       sender,
		Content:        content,
		Embedding:      embedding,
		ExpiresAt:      expiresAt,
		GroupID:        groupID,
		SourceUUID:     sourceUUID,
		IsOutgoing:     isOutgoing,
		ViewOnce:       viewOnce,
		HasAttachments: hasAttachments,
		RawJSON:        rawJSON,
	}

	messageID, err := storage.SaveMessage(ctx, record)
	if err != nil {
		log.Printf("Storage error: %v", err)
		return
	}
	if messageID == "" {
		return // duplicate
	}

	// Save attachments
	for _, att := range attachments {
		if _, err := storage.SaveAttachment(ctx, store.AttachmentRecord{
			MessageID:          messageID,
			SignalAttachmentID: att.Id,
			ContentType:        att.ContentType,
			Filename:           att.Filename,
			Size:               att.Size,
		}); err != nil {
			log.Printf("Attachment save error: %v", err)
		}
	}

	// Extract and save URLs
	if content != "" {
		urls := extract.URLs(content)
		for _, u := range urls {
			if _, err := storage.SaveURL(ctx, store.URLRecord{
				MessageID: messageID,
				URL:       u.URL,
				Domain:    u.Domain,
			}); err != nil {
				log.Printf("URL save error: %v", err)
			}
		}
	}

	// Upsert group if present
	if groupID != nil && dataMsg.GroupInfo != nil {
		_ = storage.UpsertGroup(ctx, store.GroupRecord{
			GroupID: *groupID,
		})
	}

	log.Println("Message stored.")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
