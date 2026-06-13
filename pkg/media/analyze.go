package media

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"signal-sideband/pkg/store"
)

type AnalyzeWorker struct {
	store     *store.Store
	client    *openai.Client
	model     string
	interval  time.Duration
	mediaPath string
	maxBytes  int64
}

func NewAnalyzeWorker(s *store.Store, xaiAPIKey string, interval time.Duration, mediaPath string) *AnalyzeWorker {
	cfg := openai.DefaultConfig(xaiAPIKey)
	cfg.BaseURL = "https://api.x.ai/v1"
	return &AnalyzeWorker{
		store:     s,
		client:    openai.NewClientWithConfig(cfg),
		model:     firstNonEmpty(os.Getenv("XAI_VISION_MODEL"), "grok-4.3"),
		interval:  interval,
		mediaPath: mediaPath,
		maxBytes:  int64FromEnv("XAI_VISION_MAX_IMAGE_BYTES", 5*1024*1024),
	}
}

func (w *AnalyzeWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("Analysis worker started")
	w.process(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Analysis worker stopped")
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *AnalyzeWorker) process(ctx context.Context) {
	attachments, err := w.store.GetUnanalyzedAttachments(ctx)
	if err != nil {
		log.Printf("Analysis worker: fetch error: %v", err)
		return
	}

	for _, a := range attachments {
		var imagePath string
		if strings.HasPrefix(a.ContentType, "image/") {
			imagePath = firstNonEmpty(a.ThumbnailPath, a.LocalPath)
		} else if strings.HasPrefix(a.ContentType, "video/") {
			// Use thumbnail for videos; skip if not yet generated
			if a.ThumbnailPath == "" {
				continue
			}
			imagePath = a.ThumbnailPath
		} else {
			continue
		}

		if imagePath == "" {
			continue
		}

		analysis, err := w.analyzeImage(ctx, imagePath, a.ContentType)
		if err != nil {
			log.Printf("Analysis worker: analyze %s failed: %v", a.ID, err)
			if markErr := w.store.MarkAttachmentAnalysisFailed(ctx, a.ID, err.Error()); markErr != nil {
				log.Printf("Analysis worker: mark failure %s failed: %v", a.ID, markErr)
			}
			continue
		}

		if err := w.store.MarkAttachmentAnalyzed(ctx, a.ID, analysis); err != nil {
			log.Printf("Analysis worker: save analysis %s failed: %v", a.ID, err)
			continue
		}
		log.Printf("Analysis worker: analyzed %s", a.ID)
	}
}

func (w *AnalyzeWorker) analyzeImage(ctx context.Context, imagePath, contentType string) (json.RawMessage, error) {
	info, err := os.Stat(imagePath)
	if err != nil {
		return nil, fmt.Errorf("stat image: %w", err)
	}
	if info.Size() > w.maxBytes {
		return nil, fmt.Errorf("image too large for inline vision payload: %d bytes > %d bytes", info.Size(), w.maxBytes)
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}

	// Determine mime type for the data URI
	mime := contentType
	if strings.HasPrefix(contentType, "video/") {
		mime = "image/jpeg" // thumbnails are JPEG
	}

	dataURI := fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(data))

	resp, err := w.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: w.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: `You are an image analysis assistant. Analyze the image and respond with ONLY a JSON object (no markdown, no code blocks) with these fields:
- "description": A 1-2 sentence description of what the image shows
- "text_content": Any text visible in the image (empty string if none)
- "colors": Dominant colors as a comma-separated string
- "objects": Key objects/subjects as a comma-separated string
- "scene": The type of scene (e.g. "outdoor landscape", "screenshot", "food photo", "selfie", "meme", "document")`,
			},
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    dataURI,
							Detail: openai.ImageURLDetailLow,
						},
					},
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "Analyze this image.",
					},
				},
			},
		},
		MaxTokens:   512,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("vision api: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from vision api")
	}

	content := resp.Choices[0].Message.Content

	// Strip markdown code blocks if present
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		if idx := strings.Index(content[3:], "\n"); idx != -1 {
			content = content[3+idx+1:]
		}
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
		content = strings.TrimSpace(content)
	}

	// Validate JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("invalid json response: %w (content: %s)", err, content)
	}

	// Add metadata
	parsed["model"] = w.model
	parsed["analyzed_at"] = time.Now().UTC().Format(time.RFC3339)

	result, err := json.Marshal(parsed)
	if err != nil {
		return nil, fmt.Errorf("marshal analysis: %w", err)
	}

	return json.RawMessage(result), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func int64FromEnv(name string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		log.Printf("Warning: invalid %s=%q; using %d", name, value, fallback)
		return fallback
	}
	return parsed
}
