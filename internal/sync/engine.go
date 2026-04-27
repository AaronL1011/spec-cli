// Package sync coordinates section-scoped synchronization between local specs and docs providers.
package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aaronl1011/spec-cli/internal/adapter"
	"github.com/aaronl1011/spec-cli/internal/markdown"
	"github.com/aaronl1011/spec-cli/internal/store"
)

// Sync directions.
const (
	DirectionIn   = "in"
	DirectionOut  = "out"
	DirectionBoth = "both"
)

// Conflict strategies.
const (
	ConflictWarn  = "warn"
	ConflictAbort = "abort"
	ConflictForce = "force"
	ConflictSkip  = "skip"
)

const (
	stateDirectionIn  = "in"
	stateDirectionOut = "out"
)

// ErrSyncConflict is returned when conflicts are found and the strategy is abort.
var ErrSyncConflict = errors.New("sync conflicts detected")

// Options configures one sync run.
type Options struct {
	SpecID           string
	SpecPath         string
	Direction        string
	ConflictStrategy string
	OwnerRole        string
	UserName         string
	DryRun           bool
}

// Engine coordinates sync through adapter interfaces and store APIs.
type Engine struct {
	docs adapter.DocsAdapter
	db   *store.DB
}

// NewEngine creates a sync engine.
func NewEngine(docs adapter.DocsAdapter, db *store.DB) *Engine {
	return &Engine{docs: docs, db: db}
}

// Report summarizes the outcome of a sync run.
type Report struct {
	SpecID           string
	Direction        string
	DryRun           bool
	OutboundPushed   bool
	InboundApplied   []string
	OutboundSections []string
	Conflicts        []SectionConflict
	Skipped          []SectionSkip
	Unchanged        []string
	PageMissing      bool
}

// HasBlockingConflicts returns whether conflicts should block the command.
func (r Report) HasBlockingConflicts() bool {
	return len(r.Conflicts) > 0
}

// SectionConflict describes a local/remote section conflict.
type SectionConflict struct {
	Section    string
	Owner      string
	LocalHash  string
	RemoteHash string
	Reason     string
}

// SectionSkip describes a section that was intentionally not updated.
type SectionSkip struct {
	Section string
	Reason  string
}

// Run executes a sync run.
func (e *Engine) Run(ctx context.Context, opts Options) (*Report, error) {
	if e.docs == nil {
		return nil, fmt.Errorf("docs adapter is not configured")
	}
	if opts.SpecID == "" {
		return nil, fmt.Errorf("spec id is required")
	}
	if opts.SpecPath == "" {
		return nil, fmt.Errorf("spec path is required")
	}

	direction, err := normalizeDirection(opts.Direction)
	if err != nil {
		return nil, err
	}
	strategy, err := normalizeConflictStrategy(opts.ConflictStrategy)
	if err != nil {
		return nil, err
	}
	opts.Direction = direction
	opts.ConflictStrategy = strategy

	report := &Report{SpecID: opts.SpecID, Direction: direction, DryRun: opts.DryRun}

	if direction == DirectionIn || direction == DirectionBoth {
		if err := e.runInbound(ctx, opts, report); err != nil {
			return report, err
		}
	}
	if direction == DirectionOut || direction == DirectionBoth {
		if err := e.runOutbound(ctx, opts, report); err != nil {
			return report, err
		}
	}

	e.logActivity(opts, report)
	return report, nil
}

func (e *Engine) runInbound(ctx context.Context, opts Options, report *Report) error {
	data, err := os.ReadFile(opts.SpecPath)
	if err != nil {
		return fmt.Errorf("reading local spec %s: %w", opts.SpecPath, err)
	}
	content := string(data)
	sections := markdown.ExtractSections(content)
	localBySlug := sectionsBySlug(sections)

	remoteSections, err := e.docs.FetchSections(ctx, opts.SpecID)
	if err != nil {
		return fmt.Errorf("fetching remote sections for %s: %w", opts.SpecID, err)
	}
	if remoteSections == nil {
		report.PageMissing = true
		return nil
	}

	nextContent := content
	for _, section := range sections {
		remoteContent, ok := remoteSections[section.Slug]
		if !ok {
			report.Skipped = append(report.Skipped, SectionSkip{Section: section.Slug, Reason: "remote section missing"})
			continue
		}

		localHash := Hash(section.Content)
		remoteHash := Hash(remoteContent)
		if localHash == remoteHash {
			report.Unchanged = append(report.Unchanged, section.Slug)
			if !opts.DryRun {
				e.setSectionState(opts.SpecID, section.Slug, localHash, remoteHash)
			}
			continue
		}

		lastIn := e.stateHash(opts.SpecID, section.Slug, stateDirectionIn)
		lastOut := e.stateHash(opts.SpecID, section.Slug, stateDirectionOut)
		remoteChanged := lastIn == "" || remoteHash != lastIn
		localChanged := localChangedSinceSync(section.Content, localHash, lastOut, lastIn)

		if !remoteChanged {
			report.Unchanged = append(report.Unchanged, section.Slug)
			continue
		}
		if ownerMismatch(section, opts.OwnerRole) && opts.ConflictStrategy != ConflictForce {
			report.Skipped = append(report.Skipped, SectionSkip{
				Section: section.Slug,
				Reason:  fmt.Sprintf("owned by %s", section.Owner),
			})
			continue
		}
		if localChanged && opts.ConflictStrategy != ConflictForce {
			conflict := SectionConflict{
				Section:    section.Slug,
				Owner:      section.Owner,
				LocalHash:  localHash,
				RemoteHash: remoteHash,
				Reason:     "local and remote changed since last sync",
			}
			report.Conflicts = append(report.Conflicts, conflict)
			if opts.ConflictStrategy == ConflictAbort {
				continue
			}
			report.Skipped = append(report.Skipped, SectionSkip{Section: section.Slug, Reason: "conflict"})
			continue
		}

		if _, ok := localBySlug[section.Slug]; ok {
			nextContent, err = markdown.ReplaceSectionContent(nextContent, section.Slug, remoteContent)
			if err != nil {
				return fmt.Errorf("applying inbound section %s: %w", section.Slug, err)
			}
			report.InboundApplied = append(report.InboundApplied, section.Slug)
			if !opts.DryRun {
				e.setSectionState(opts.SpecID, section.Slug, Hash(remoteContent), Hash(remoteContent))
			}
		}
	}

	if opts.ConflictStrategy == ConflictAbort && report.HasBlockingConflicts() {
		return ErrSyncConflict
	}
	if !opts.DryRun && nextContent != content {
		if err := os.WriteFile(opts.SpecPath, []byte(nextContent), 0o644); err != nil {
			return fmt.Errorf("writing local spec %s: %w", opts.SpecPath, err)
		}
	}
	return nil
}

func (e *Engine) runOutbound(ctx context.Context, opts Options, report *Report) error {
	data, err := os.ReadFile(opts.SpecPath)
	if err != nil {
		return fmt.Errorf("reading local spec %s: %w", opts.SpecPath, err)
	}
	content := string(data)
	sections := markdown.ExtractSections(content)
	for _, section := range sections {
		report.OutboundSections = append(report.OutboundSections, section.Slug)
	}

	if !opts.DryRun {
		if err := e.docs.PushFull(ctx, opts.SpecID, content); err != nil {
			return fmt.Errorf("pushing %s to docs provider: %w", opts.SpecID, err)
		}
		for _, section := range sections {
			hash := Hash(section.Content)
			e.setSectionState(opts.SpecID, section.Slug, hash, hash)
		}
	}
	report.OutboundPushed = true
	return nil
}

// Hash returns the deterministic sync hash for section content.
func Hash(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

func (e *Engine) stateHash(specID, section, direction string) string {
	if e.db == nil {
		return ""
	}
	entry, err := e.db.SyncStateGet(specID, section, direction)
	if err != nil || entry == nil {
		return ""
	}
	return entry.Hash
}

func (e *Engine) setSectionState(specID, section, localHash, remoteHash string) {
	if e.db == nil {
		return
	}
	_ = e.db.SyncStateSet(specID, section, stateDirectionOut, localHash)
	_ = e.db.SyncStateSet(specID, section, stateDirectionIn, remoteHash)
}

func (e *Engine) logActivity(opts Options, report *Report) {
	if e.db == nil || report == nil || opts.DryRun {
		return
	}
	metadata, err := json.Marshal(map[string]interface{}{
		"direction":         report.Direction,
		"inbound_applied":   report.InboundApplied,
		"outbound_pushed":   report.OutboundPushed,
		"outbound_sections": len(report.OutboundSections),
		"conflicts":         len(report.Conflicts),
		"skipped":           len(report.Skipped),
		"page_missing":      report.PageMissing,
	})
	if err != nil {
		return
	}
	user := opts.UserName
	if user == "" {
		user = "system"
	}
	_ = e.db.ActivityLog(opts.SpecID, "sync", fmt.Sprintf("synced %s", report.Direction), string(metadata), user)
}

func sectionsBySlug(sections []markdown.Section) map[string]markdown.Section {
	bySlug := make(map[string]markdown.Section, len(sections))
	for _, section := range sections {
		bySlug[section.Slug] = section
	}
	return bySlug
}

func localChangedSinceSync(content, localHash, lastOut, lastIn string) bool {
	if strings.TrimSpace(content) == "" {
		return false
	}
	switch {
	case lastOut != "":
		return localHash != lastOut
	case lastIn != "":
		return localHash != lastIn
	default:
		return true
	}
}

func ownerMismatch(section markdown.Section, ownerRole string) bool {
	if section.Owner == "" || ownerRole == "" {
		return false
	}
	return !strings.EqualFold(section.Owner, ownerRole)
}

func normalizeDirection(direction string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "", DirectionBoth:
		return DirectionBoth, nil
	case DirectionIn, "inbound":
		return DirectionIn, nil
	case DirectionOut, "outbound":
		return DirectionOut, nil
	default:
		return "", fmt.Errorf("invalid sync direction %q — use in, out, or both", direction)
	}
}

func normalizeConflictStrategy(strategy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "", ConflictWarn:
		return ConflictWarn, nil
	case ConflictAbort:
		return ConflictAbort, nil
	case ConflictForce:
		return ConflictForce, nil
	case ConflictSkip:
		return ConflictSkip, nil
	default:
		return "", fmt.Errorf("invalid sync conflict strategy %q — use warn, abort, force, or skip", strategy)
	}
}
