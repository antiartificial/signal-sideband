package ai

type Embedder interface {
	Embed(text string) ([]float32, error)
}

type MockEmbedder struct{}

func (m *MockEmbedder) Embed(text string) ([]float32, error) {
	// Return a zero vector of 1536 dimensions for testing
	return make([]float32, 1536), nil
}
