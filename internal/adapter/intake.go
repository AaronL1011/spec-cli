package adapter

import "context"

// IntakeAdapter manages external intake source integration.
type IntakeAdapter interface {
	// FetchItems returns new intake items from the external source.
	FetchItems(ctx context.Context) ([]IntakeItem, error)
}

// IntakeItem represents an item from an external intake source.
type IntakeItem struct {
	Title     string
	Source    string
	SourceRef string
	Priority  string
	Body      string
}
