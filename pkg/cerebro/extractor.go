package cerebro

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

type Extractor struct {
	store    *store.Store
	provider llm.Provider
}

func NewExtractor(s *store.Store, p llm.Provider) *Extractor {
	return &Extractor{store: s, provider: p}
}

type extractionResult struct {
	Concepts []extractedConcept `json:"concepts"`
	Edges    []extractedEdge    `json:"edges"`
}

type extractedConcept struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type extractedEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

const extractionSystemPrompt = `You are a knowledge graph extraction engine. Given a chat transcript, extract concepts and their relationships.

For each concept, provide:
- name: a concise, canonical name (lowercase unless proper noun)
- category: one of "topic", "person", "place", "media", "event", "idea"
- description: a one-sentence description based on context

For each edge/relationship, provide:
- source: concept name (must match a concept name exactly)
- target: concept name (must match a concept name exactly)
- relation: a short relationship label like "discussed_by", "related_to", "mentioned_in", "created_by", "located_in", "part_of"

Focus on meaningful, recurring concepts — not every noun. Aim for 5-20 concepts per batch.

Respond ONLY with JSON in this exact format:
{
  "concepts": [{"name": "...", "category": "...", "description": "..."}],
  "edges": [{"source": "...", "target": "...", "relation": "..."}]
}`

func (e *Extractor) Extract(ctx context.Context, start, end time.Time) (*store.CerebroExtraction, error) {
	messages, err := e.store.GetMessagesByTimeRange(ctx, start, end, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch messages: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in time range")
	}

	// Format messages for LLM
	var sb strings.Builder
	for _, m := range messages {
		sender := m.SenderID
		if m.IsOutgoing {
			sender = "me"
		}
		ts := m.CreatedAt.Format("15:04")
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", ts, sender, m.Content))
	}

	resp, err := e.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: extractionSystemPrompt,
		UserPrompt:   fmt.Sprintf("Extract concepts and relationships from this chat transcript (%d messages):\n\n%s", len(messages), sb.String()),
		MaxTokens:    4096,
		Temperature:  0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	// Parse response - strip markdown code blocks if present
	content := resp.Content
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

	var result extractionResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &result); err != nil {
		return nil, fmt.Errorf("parse extraction result: %w (response: %s)", err, truncate(resp.Content, 200))
	}

	// Upsert concepts and build name->id map
	nameToID := make(map[string]string)
	conceptCount := 0
	for _, c := range result.Concepts {
		validCategories := map[string]bool{"topic": true, "person": true, "place": true, "media": true, "event": true, "idea": true}
		if !validCategories[c.Category] {
			c.Category = "topic"
		}
		id, err := e.store.UpsertConcept(ctx, store.CerebroConcept{
			Name:        c.Name,
			Category:    c.Category,
			Description: c.Description,
			LastSeen:    end,
		})
		if err != nil {
			log.Printf("cerebro: upsert concept %q failed: %v", c.Name, err)
			continue
		}
		nameToID[c.Name] = id
		conceptCount++
	}

	// Upsert edges
	edgeCount := 0
	for _, edge := range result.Edges {
		sourceID, ok1 := nameToID[edge.Source]
		targetID, ok2 := nameToID[edge.Target]
		if !ok1 || !ok2 {
			continue
		}
		if _, err := e.store.UpsertEdge(ctx, store.CerebroEdge{
			SourceID: sourceID,
			TargetID: targetID,
			Relation: edge.Relation,
		}); err != nil {
			log.Printf("cerebro: upsert edge %q->%q failed: %v", edge.Source, edge.Target, err)
			continue
		}
		edgeCount++
	}

	// Log extraction
	extraction := store.CerebroExtraction{
		BatchStart:   start,
		BatchEnd:     end,
		MessageCount: len(messages),
		ConceptCount: conceptCount,
		EdgeCount:    edgeCount,
		LLMProvider:  e.provider.Name(),
		LLMModel:     resp.Model,
		TokenCount:   resp.InputTokens + resp.OutputTokens,
	}
	id, err := e.store.SaveExtraction(ctx, extraction)
	if err != nil {
		return nil, fmt.Errorf("save extraction: %w", err)
	}
	extraction.ID = id

	log.Printf("cerebro: extracted %d concepts, %d edges from %d messages", conceptCount, edgeCount, len(messages))
	return &extraction, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
