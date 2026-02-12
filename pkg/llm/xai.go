package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type XAIProvider struct {
	client *openai.Client
	model  string
}

func NewXAIProvider(apiKey string) *XAIProvider {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = "https://api.x.ai/v1"
	return &XAIProvider{
		client: openai.NewClientWithConfig(cfg),
		model:  "grok-3-mini-fast",
	}
}

func (x *XAIProvider) Name() string { return "xai" }

func (x *XAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
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

	resp, err := x.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       x.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: float32(req.Temperature),
	})
	if err != nil {
		return nil, fmt.Errorf("xai completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from xai")
	}

	return &CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		Model:        resp.Model,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}, nil
}
