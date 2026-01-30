package ai

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIEmbedder struct {
	client *openai.Client
}

func NewOpenAIEmbedder(apiKey string) *OpenAIEmbedder {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAIEmbedder{
		client: openai.NewClient(apiKey),
	}
}

func (o *OpenAIEmbedder) Embed(text string) ([]float32, error) {
	ctx := context.Background()
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	}

	resp, err := o.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai embedding error: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("openai returned empty embedding data")
	}

	return resp.Data[0].Embedding, nil
}
