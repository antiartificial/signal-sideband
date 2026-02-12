package digest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"signal-sideband/pkg/llm"
	"signal-sideband/pkg/media"
	"signal-sideband/pkg/store"
)

type InsightsGenerator struct {
	store    *store.Store
	provider llm.Provider
	picGen   *media.PicOfDayGenerator
}

func NewInsightsGenerator(s *store.Store, p llm.Provider, picGen *media.PicOfDayGenerator) *InsightsGenerator {
	return &InsightsGenerator{store: s, provider: p, picGen: picGen}
}

type insightsJSON struct {
	Overview   string   `json:"overview"`
	Themes     []string `json:"themes"`
	QuoteIndex int      `json:"quote_index"`
}

func (g *InsightsGenerator) GenerateDailyInsights(ctx context.Context) error {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := now

	// If early in the day, also include yesterday
	if now.Hour() < 6 {
		start = start.Add(-24 * time.Hour)
	}

	messages, err := g.store.GetMessagesByTimeRange(ctx, start, end, nil)
	if err != nil {
		return fmt.Errorf("fetch messages: %w", err)
	}

	if len(messages) < 5 {
		log.Println("Insights: fewer than 5 messages, skipping")
		return nil
	}

	// Format messages for LLM
	var sb strings.Builder
	for i, m := range messages {
		sender := m.SenderID
		if m.IsOutgoing {
			sender = "me"
		}
		ts := m.CreatedAt.Format("15:04")
		sb.WriteString(fmt.Sprintf("[%d] [%s] %s: %s\n", i, ts, sender, m.Content))
	}

	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: `You are analyzing a day's chat messages. Respond with ONLY a JSON object (no markdown, no code blocks):
{
  "overview": "A 2-3 sentence conversational gist of the day. Start with 'Today the group...' or similar casual phrasing.",
  "themes": ["theme1", "theme2", "theme3"],
  "quote_index": 0
}
- themes: 3-5 topic/theme tags (short, lowercase)
- quote_index: the [index] of the most interesting, funny, or notable message`,
		UserPrompt:  sb.String(),
		MaxTokens:   512,
		Temperature: 0.4,
	})
	if err != nil {
		return fmt.Errorf("llm completion: %w", err)
	}

	content := resp.Content
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

	var parsed insightsJSON
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return fmt.Errorf("parse insights json: %w (content: %s)", err, content)
	}

	// Pick the quote
	var quoteContent, quoteSender string
	if parsed.QuoteIndex >= 0 && parsed.QuoteIndex < len(messages) {
		msg := messages[parsed.QuoteIndex]
		quoteContent = msg.Content
		quoteSender = msg.SenderID
		if msg.IsOutgoing {
			quoteSender = "me"
		}
	} else if len(messages) > 0 {
		// Fallback to random quote
		qc, qs, err := g.store.GetRandomQuote(ctx)
		if err == nil {
			quoteContent = qc
			quoteSender = qs
		}
	}

	themes, _ := json.Marshal(parsed.Themes)

	id, err := g.store.SaveDailyInsight(ctx, parsed.Overview, themes, quoteContent, quoteSender)
	if err != nil {
		return fmt.Errorf("save insight: %w", err)
	}

	log.Printf("Insights: generated daily insight %s", id)

	// Generate nano banana picture of the day
	if g.picGen != nil {
		imagePath, err := g.picGen.Generate(ctx, parsed.Themes, parsed.Overview)
		if err != nil {
			log.Printf("PicOfDay: generation failed: %v", err)
		} else if imagePath != "" {
			if err := g.store.SetInsightImagePath(ctx, id, imagePath); err != nil {
				log.Printf("PicOfDay: failed to save image path: %v", err)
			}
		}
	}

	return nil
}
