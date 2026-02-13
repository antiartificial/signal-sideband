package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type PerplexityProvider struct {
	client *openai.Client
	model  string
}

func NewPerplexityProvider(apiKey string) *PerplexityProvider {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = "https://api.perplexity.ai"
	return &PerplexityProvider{
		client: openai.NewClientWithConfig(cfg),
		model:  "sonar",
	}
}

func (p *PerplexityProvider) Name() string { return "perplexity" }

func (p *PerplexityProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	messages := []openai.ChatCompletionMessage{}
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.UserPrompt,
	})

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: float32(req.Temperature),
	})
	if err != nil {
		return nil, fmt.Errorf("perplexity completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from perplexity")
	}

	return &CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		Model:        resp.Model,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}, nil
}
