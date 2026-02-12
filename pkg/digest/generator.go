package digest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"signal-sideband/pkg/llm"
	"signal-sideband/pkg/store"
)

type Generator struct {
	store    *store.Store
	provider llm.Provider
}

func NewGenerator(s *store.Store, p llm.Provider) *Generator {
	return &Generator{store: s, provider: p}
}

type digestJSON struct {
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Topics      []string `json:"topics"`
	Decisions   []string `json:"decisions"`
	ActionItems []string `json:"action_items"`
}

func (g *Generator) Generate(ctx context.Context, start, end time.Time, groupID *string, lens ...string) (*store.DigestRecord, error) {
	messages, err := g.store.GetMessagesByTimeRange(ctx, start, end, groupID)
	if err != nil {
		return nil, fmt.Errorf("fetch messages: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in the specified time range")
	}

	// Format messages for the LLM
	var sb strings.Builder
	for _, m := range messages {
		sender := m.SenderID
		if m.IsOutgoing {
			sender = "me"
		}
		ts := m.CreatedAt.Format("15:04")
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", ts, sender, m.Content))
	}

	periodLabel := fmt.Sprintf("%s to %s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006"))

	selectedLens := ""
	if len(lens) > 0 {
		selectedLens = lens[0]
	}
	temperature := 0.3
	if selectedLens != "" && selectedLens != "default" {
		temperature = 0.7 // more creative for character lenses
	}

	// Call LLM
	resp, err := g.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: systemPromptForLens(selectedLens),
		UserPrompt:   buildUserPrompt(sb.String(), periodLabel),
		MaxTokens:    4096,
		Temperature:  temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	// Parse response
	var parsed digestJSON
	content := resp.Content

	// Try to extract JSON from markdown code blocks if present
	if idx := strings.Index(content, "```json"); idx != -1 {
		content = content[idx+7:]
		if end := strings.Index(content, "```"); end != -1 {
			content = content[:end]
		}
	} else if idx := strings.Index(content, "```"); idx != -1 {
		content = content[idx+3:]
		if end := strings.Index(content, "```"); end != -1 {
			content = content[:end]
		}
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &parsed); err != nil {
		log.Printf("Failed to parse LLM response as JSON, using raw content: %v", err)
		parsed = digestJSON{
			Title:   periodLabel + " Digest",
			Summary: resp.Content,
		}
	}

	topics, _ := json.Marshal(parsed.Topics)
	decisions, _ := json.Marshal(parsed.Decisions)
	actionItems, _ := json.Marshal(parsed.ActionItems)

	record := store.DigestRecord{
		Title:       parsed.Title,
		Summary:     parsed.Summary,
		Topics:      topics,
		Decisions:   decisions,
		ActionItems: actionItems,
		PeriodStart: start,
		PeriodEnd:   end,
		GroupID:     groupID,
		LLMProvider: g.provider.Name(),
		LLMModel:    resp.Model,
		TokenCount:  resp.InputTokens + resp.OutputTokens,
	}

	id, err := g.store.SaveDigest(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("save digest: %w", err)
	}
	record.ID = id

	return &record, nil
}
