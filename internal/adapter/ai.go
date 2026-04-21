package adapter

import "context"

// AIAdapter manages LLM integration for drafting, summarisation, and embedding.
type AIAdapter interface {
	// Complete sends a prompt and returns the completion.
	Complete(ctx context.Context, prompt string, system string) (string, error)
	// Embed returns a vector embedding for the given text.
	// Returns nil, ErrEmbeddingsNotSupported if the provider doesn't support embeddings.
	Embed(ctx context.Context, text string) ([]float32, error)
}
