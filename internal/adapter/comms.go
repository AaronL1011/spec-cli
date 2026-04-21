package adapter

import (
	"context"
	"time"
)

// CommsAdapter sends notifications and retrieves mentions from a comms platform.
type CommsAdapter interface {
	// Notify sends a structured message to the configured channel.
	Notify(ctx context.Context, msg Notification) error
	// PostStandup posts a formatted standup to the standup channel.
	PostStandup(ctx context.Context, standup StandupReport) error
	// FetchMentions returns recent mentions of spec IDs in configured channels.
	FetchMentions(ctx context.Context, since time.Time) ([]Mention, error)
}
