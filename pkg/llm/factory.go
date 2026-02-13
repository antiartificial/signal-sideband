package llm

import (
	"fmt"
	"os"
)

func NewProvider(name string) (Provider, error) {
	switch name {
	case "claude", "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return NewClaudeProvider(key), nil
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		return NewOpenAIProvider(key), nil
	case "xai", "grok":
		key := os.Getenv("XAI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("XAI_API_KEY not set")
		}
		return NewXAIProvider(key), nil
	case "perplexity":
		key := os.Getenv("PERPLEXITY_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("PERPLEXITY_API_KEY not set")
		}
		return NewPerplexityProvider(key), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", name)
	}
}
