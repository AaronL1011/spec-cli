package effects

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aaronl1011/spec-cli/internal/adapter"
	"github.com/aaronl1011/spec-cli/internal/markdown"
	"github.com/aaronl1011/spec-cli/internal/store"
)

// NotifierAdapter bridges adapter.CommsAdapter to the Notifier interface.
type NotifierAdapter struct {
	Comms  adapter.CommsAdapter
	SpecID string
	Title  string
}

func (n *NotifierAdapter) Notify(ctx context.Context, target, message string) error {
	return n.Comms.Notify(ctx, adapter.Notification{
		SpecID:  n.SpecID,
		Title:   n.Title,
		Message: message,
		Mention: target,
	})
}

// WebhookerAdapter implements the Webhooker interface using net/http.
type WebhookerAdapter struct {
	Client *http.Client
}

func (w *WebhookerAdapter) Call(ctx context.Context, url string, payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling webhook payload: %w", err)
	}

	client := w.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("calling webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// LoggerAdapter bridges store.DB and markdown decision log to the Logger interface.
type LoggerAdapter struct {
	DB      *store.DB
	SpecDir string
	SpecID  string
}

func (l *LoggerAdapter) LogDecision(ctx context.Context, specID, decision string) error {
	if l.SpecDir == "" {
		return nil
	}
	path := l.SpecDir + "/" + specID + ".md"
	_, err := markdown.AppendDecision(path, decision, "system")
	return err
}

func (l *LoggerAdapter) LogEvent(ctx context.Context, specID, event string, data map[string]interface{}) error {
	if l.DB == nil {
		return nil
	}
	meta, _ := json.Marshal(data)
	return l.DB.ActivityLog(specID, event, event, string(meta), "system")
}
