package adapter

import "context"

// PMAdapter manages project management tool integration.
type PMAdapter interface {
	// CreateEpic creates a new epic/issue linked to a spec.
	CreateEpic(ctx context.Context, spec SpecMeta) (epicKey string, err error)
	// UpdateStatus syncs the spec's pipeline status to the PM tool.
	UpdateStatus(ctx context.Context, epicKey string, status string) error
	// FetchUpdates returns status changes from the PM tool since last sync.
	FetchUpdates(ctx context.Context, epicKey string) (*PMUpdate, error)
}
