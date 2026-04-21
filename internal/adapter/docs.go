package adapter

import "context"

// DocsAdapter manages documentation tool integration.
type DocsAdapter interface {
	// FetchSections retrieves the current content of the spec from the docs provider,
	// keyed by section slug.
	FetchSections(ctx context.Context, specID string) (map[string]string, error)
	// PushFull publishes the complete spec to the docs provider.
	PushFull(ctx context.Context, specID string, content string) error
	// PageURL returns the URL of the spec's page in the docs provider.
	PageURL(ctx context.Context, specID string) (string, error)
}
