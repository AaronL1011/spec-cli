package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aaronl1011/spec-cli/internal/store"
)

type fakeDocs struct {
	sections      map[string]string
	pushedSpecID  string
	pushedContent string
}

func (f *fakeDocs) FetchSections(ctx context.Context, specID string) (map[string]string, error) {
	return f.sections, nil
}

func (f *fakeDocs) PushFull(ctx context.Context, specID string, content string) error {
	f.pushedSpecID = specID
	f.pushedContent = content
	return nil
}

func (f *fakeDocs) PageURL(ctx context.Context, specID string) (string, error) {
	return "", nil
}

func TestRun_Outbound_PushesFullSpecAndRecordsHashes(t *testing.T) {
	specPath := writeSpec(t, `---
id: SPEC-001
title: Test
status: draft
created: 2026-01-01
updated: 2026-01-01
---

## Problem Statement <!-- owner: tl -->
Local content
`)
	db := openMemoryDB(t)
	docs := &fakeDocs{}

	report, err := NewEngine(docs, db).Run(context.Background(), Options{
		SpecID:    "SPEC-001",
		SpecPath:  specPath,
		Direction: DirectionOut,
		UserName:  "alice",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.OutboundPushed {
		t.Fatal("Run() OutboundPushed = false")
	}
	if docs.pushedSpecID != "SPEC-001" || docs.pushedContent == "" {
		t.Fatalf("PushFull() spec/content = %q/%q", docs.pushedSpecID, docs.pushedContent)
	}
	state, err := db.SyncStateGet("SPEC-001", "problem_statement", "out")
	if err != nil {
		t.Fatalf("SyncStateGet() error = %v", err)
	}
	if state == nil || state.Hash != Hash("Local content") {
		t.Fatalf("stored hash = %#v, want local hash", state)
	}
}

func TestRun_Inbound_AppliesRemoteWhenLocalUnchanged(t *testing.T) {
	specPath := writeSpec(t, `---
id: SPEC-001
title: Test
status: draft
created: 2026-01-01
updated: 2026-01-01
---

## Problem Statement <!-- owner: tl -->
Old content
`)
	db := openMemoryDB(t)
	if err := db.SyncStateSet("SPEC-001", "problem_statement", "out", Hash("Old content")); err != nil {
		t.Fatalf("SyncStateSet(out) error = %v", err)
	}
	if err := db.SyncStateSet("SPEC-001", "problem_statement", "in", Hash("Old content")); err != nil {
		t.Fatalf("SyncStateSet(in) error = %v", err)
	}
	docs := &fakeDocs{sections: map[string]string{"problem_statement": "Remote content"}}

	report, err := NewEngine(docs, db).Run(context.Background(), Options{
		SpecID:    "SPEC-001",
		SpecPath:  specPath,
		Direction: DirectionIn,
		OwnerRole: "tl",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(report.InboundApplied) != 1 || report.InboundApplied[0] != "problem_statement" {
		t.Fatalf("InboundApplied = %#v", report.InboundApplied)
	}
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := string(data); !contains(got, "Remote content") {
		t.Fatalf("updated spec = %q, want remote content", got)
	}
}

func TestRun_Inbound_AbortOnConflict(t *testing.T) {
	specPath := writeSpec(t, `---
id: SPEC-001
title: Test
status: draft
created: 2026-01-01
updated: 2026-01-01
---

## Problem Statement <!-- owner: tl -->
Local edit
`)
	db := openMemoryDB(t)
	if err := db.SyncStateSet("SPEC-001", "problem_statement", "out", Hash("Base content")); err != nil {
		t.Fatalf("SyncStateSet(out) error = %v", err)
	}
	if err := db.SyncStateSet("SPEC-001", "problem_statement", "in", Hash("Base content")); err != nil {
		t.Fatalf("SyncStateSet(in) error = %v", err)
	}
	docs := &fakeDocs{sections: map[string]string{"problem_statement": "Remote edit"}}

	report, err := NewEngine(docs, db).Run(context.Background(), Options{
		SpecID:           "SPEC-001",
		SpecPath:         specPath,
		Direction:        DirectionIn,
		ConflictStrategy: ConflictAbort,
		OwnerRole:        "tl",
	})
	if err != ErrSyncConflict {
		t.Fatalf("Run() error = %v, want ErrSyncConflict", err)
	}
	if report == nil || len(report.Conflicts) != 1 {
		t.Fatalf("conflicts = %#v, want one conflict", report)
	}
}

func TestRun_Inbound_SkipsOwnedSectionForOtherRole(t *testing.T) {
	specPath := writeSpec(t, `---
id: SPEC-001
title: Test
status: draft
created: 2026-01-01
updated: 2026-01-01
---

## Acceptance Criteria <!-- owner: tl -->
Old content
`)
	db := openMemoryDB(t)
	if err := db.SyncStateSet("SPEC-001", "acceptance_criteria", "out", Hash("Old content")); err != nil {
		t.Fatalf("SyncStateSet(out) error = %v", err)
	}
	if err := db.SyncStateSet("SPEC-001", "acceptance_criteria", "in", Hash("Old content")); err != nil {
		t.Fatalf("SyncStateSet(in) error = %v", err)
	}
	docs := &fakeDocs{sections: map[string]string{"acceptance_criteria": "Remote content"}}

	report, err := NewEngine(docs, db).Run(context.Background(), Options{
		SpecID:    "SPEC-001",
		SpecPath:  specPath,
		Direction: DirectionIn,
		OwnerRole: "engineer",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(report.Skipped) != 1 || report.Skipped[0].Reason != "owned by tl" {
		t.Fatalf("Skipped = %#v", report.Skipped)
	}
}

func openMemoryDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func writeSpec(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "SPEC-001.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && index(s, substr) >= 0)
}

func index(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
