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

type Enricher struct {
	store              *store.Store
	perplexityProvider llm.Provider
	grokProvider       llm.Provider
}

func NewEnricher(s *store.Store, perplexity llm.Provider, grok llm.Provider) *Enricher {
	return &Enricher{store: s, perplexityProvider: perplexity, grokProvider: grok}
}

const perplexityPrompt = `Given this concept from a group chat knowledge graph, provide enrichment information.

Concept: %s
Category: %s
Description: %s

Respond ONLY with JSON:
{
  "summary": "A concise 2-3 sentence summary of this concept",
  "related_topics": ["topic1", "topic2", "topic3"],
  "key_facts": ["fact1", "fact2", "fact3"],
  "suggested_exploration": ["question or angle to explore further"]
}`

const grokXPrompt = `Given this concept, find relevant trending or recent discussions about it on X/Twitter.

Concept: %s
Category: %s

Respond ONLY with JSON:
{
  "trending_posts": [
    {"summary": "brief summary of a relevant trending post/discussion", "context": "why this is relevant"}
  ],
  "sentiment": "overall sentiment around this topic (positive/negative/neutral/mixed)",
  "trending_score": "low/medium/high - how much this is being discussed"
}`

const grokBooksPrompt = `Given this concept, suggest relevant books, articles, or resources.

Concept: %s
Category: %s
Description: %s

Respond ONLY with JSON:
{
  "books": [
    {"title": "Book Title", "author": "Author Name", "relevance": "why this is relevant"}
  ],
  "articles": [
    {"title": "Article Title", "source": "Source", "relevance": "why this is relevant"}
  ]
}`

func (e *Enricher) EnrichConcept(ctx context.Context, concept store.CerebroConcept) error {
	if e.perplexityProvider != nil {
		if err := e.enrichWithPerplexity(ctx, concept); err != nil {
			log.Printf("cerebro: perplexity enrichment failed for %q: %v", concept.Name, err)
		}
	}

	if e.grokProvider != nil {
		if err := e.enrichWithGrokX(ctx, concept); err != nil {
			log.Printf("cerebro: grok X enrichment failed for %q: %v", concept.Name, err)
		}
		if err := e.enrichWithGrokBooks(ctx, concept); err != nil {
			log.Printf("cerebro: grok books enrichment failed for %q: %v", concept.Name, err)
		}
	}

	return nil
}

func (e *Enricher) enrichWithPerplexity(ctx context.Context, concept store.CerebroConcept) error {
	resp, err := e.perplexityProvider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: "You are a knowledge enrichment engine. Provide factual, concise information.",
		UserPrompt:   fmt.Sprintf(perplexityPrompt, concept.Name, concept.Category, concept.Description),
		MaxTokens:    2048,
		Temperature:  0.3,
	})
	if err != nil {
		return err
	}

	content := parseJSONResponse(resp.Content)
	contentJSON, err := json.Marshal(json.RawMessage(content))
	if err != nil {
		contentJSON = []byte(`{"raw": ` + fmt.Sprintf("%q", resp.Content) + `}`)
	}

	_, err = e.store.SaveEnrichment(ctx, store.CerebroEnrichment{
		ConceptID: concept.ID,
		Source:    "perplexity",
		Content:   contentJSON,
	})
	return err
}

func (e *Enricher) enrichWithGrokX(ctx context.Context, concept store.CerebroConcept) error {
	resp, err := e.grokProvider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: "You are a social media trends analyst. Provide relevant X/Twitter trending info.",
		UserPrompt:   fmt.Sprintf(grokXPrompt, concept.Name, concept.Category),
		MaxTokens:    2048,
		Temperature:  0.5,
	})
	if err != nil {
		return err
	}

	content := parseJSONResponse(resp.Content)
	contentJSON, err := json.Marshal(json.RawMessage(content))
	if err != nil {
		contentJSON = []byte(`{"raw": ` + fmt.Sprintf("%q", resp.Content) + `}`)
	}

	ttl := 24 * time.Hour
	expiresAt := time.Now().Add(ttl)

	_, err = e.store.SaveEnrichment(ctx, store.CerebroEnrichment{
		ConceptID: concept.ID,
		Source:    "grok_x",
		Content:   contentJSON,
		ExpiresAt: &expiresAt,
	})
	return err
}

func (e *Enricher) enrichWithGrokBooks(ctx context.Context, concept store.CerebroConcept) error {
	resp, err := e.grokProvider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: "You are a research librarian. Suggest relevant books and articles.",
		UserPrompt:   fmt.Sprintf(grokBooksPrompt, concept.Name, concept.Category, concept.Description),
		MaxTokens:    2048,
		Temperature:  0.3,
	})
	if err != nil {
		return err
	}

	content := parseJSONResponse(resp.Content)
	contentJSON, err := json.Marshal(json.RawMessage(content))
	if err != nil {
		contentJSON = []byte(`{"raw": ` + fmt.Sprintf("%q", resp.Content) + `}`)
	}

	ttl := 24 * time.Hour
	expiresAt := time.Now().Add(ttl)

	_, err = e.store.SaveEnrichment(ctx, store.CerebroEnrichment{
		ConceptID: concept.ID,
		Source:    "grok_books",
		Content:   contentJSON,
		ExpiresAt: &expiresAt,
	})
	return err
}

func (e *Enricher) EnrichBatch(ctx context.Context, limit int) error {
	concepts, err := e.store.GetConceptsNeedingEnrichment(ctx, limit)
	if err != nil {
		return fmt.Errorf("get concepts needing enrichment: %w", err)
	}
	for _, c := range concepts {
		if err := e.EnrichConcept(ctx, c); err != nil {
			log.Printf("cerebro: batch enrichment failed for %q: %v", c.Name, err)
		}
	}
	if len(concepts) > 0 {
		log.Printf("cerebro: enriched %d concepts", len(concepts))
	}
	return nil
}

func parseJSONResponse(content string) string {
	trimmed := content
	if idx := strings.Index(trimmed, "```json"); idx != -1 {
		trimmed = trimmed[idx+7:]
		if end := strings.Index(trimmed, "```"); end != -1 {
			trimmed = trimmed[:end]
		}
	} else if idx := strings.Index(trimmed, "```"); idx != -1 {
		trimmed = trimmed[idx+3:]
		if end := strings.Index(trimmed, "```"); end != -1 {
			trimmed = trimmed[:end]
		}
	}
	trimmed = strings.TrimSpace(trimmed)
	var js json.RawMessage
	if json.Unmarshal([]byte(trimmed), &js) == nil {
		return trimmed
	}
	return content
}
