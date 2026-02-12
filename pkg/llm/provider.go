package llm

import "context"

type CompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
	Temperature  float64
}

type CompletionResponse struct {
	Content     string
	Model       string
	InputTokens int
	OutputTokens int
}

type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Name() string
}
