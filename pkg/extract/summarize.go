package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"signal-sideband/pkg/llm"
)

type urlSummaryResponse struct {
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
}

func isTwitterDomain(domain string) bool {
	d := strings.ToLower(domain)
	return d == "x.com" || d == "twitter.com" ||
		strings.HasSuffix(d, ".x.com") || strings.HasSuffix(d, ".twitter.com")
}

func SummarizeURL(ctx context.Context, provider llm.Provider, url, domain, title, description, sharedBy string) (summary string, tags []string, err error) {
	var userPrompt string
	sharedByCtx := ""
	if sharedBy != "" {
		sharedByCtx = fmt.Sprintf(" Shared by %s.", sharedBy)
	}

	if isTwitterDomain(domain) {
		userPrompt = fmt.Sprintf("Summarize the content at this URL: %s%s\n\nReturn a one-sentence summary and 2-4 topic tags.", url, sharedByCtx)
	} else if title != "" || description != "" {
		userPrompt = fmt.Sprintf("Given this link titled '%s' described as '%s' at %s.%s\n\nReturn a one-sentence summary and 2-4 topic tags.", title, description, url, sharedByCtx)
	} else {
		userPrompt = fmt.Sprintf("Summarize the content at this URL: %s%s\n\nReturn a one-sentence summary and 2-4 topic tags.", url, sharedByCtx)
	}

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: "You summarize web links. Include who shared it when provided. Respond with ONLY valid JSON in this exact format: {\"summary\": \"one sentence summary\", \"tags\": [\"tag1\", \"tag2\"]}. Tags should describe the content topic, not the domain. No other text.",
		UserPrompt:   userPrompt,
		MaxTokens:    256,
		Temperature:  0.3,
	})
	if err != nil {
		return "", nil, fmt.Errorf("llm completion: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	// Strip markdown code fences if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result urlSummaryResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return "", nil, fmt.Errorf("parse llm response: %w", err)
	}

	return result.Summary, result.Tags, nil
}
