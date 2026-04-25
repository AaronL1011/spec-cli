# Pilot Must-Haves Implementation Plan & Progress

## Context

The spec CLI has 5 gaps that would block an effective pilot for a tech lead managing a team using MS Teams, Jira, Confluence, and GitHub. The adapters for all four tools already exist — the gaps are in wiring, engine logic, and gate enforcement. This plan addresses them in dependency order.

**Critical prerequisite discovered**: The `activity` table only receives `build` events (from `internal/build/session.go:134`). Commands `advance`, `revert`, `eject`, `resume`, and `decide` never call `db.ActivityLog()`. Metrics and richer standup both depend on this data.

---

## Implementation Order & Status

```
Phase 0: Activity logging in commands           ✅ COMPLETE
Phase 1: Duration & link_exists gates            ✅ COMPLETE
Phase 2: Effects adapter wiring                  ✅ COMPLETE
Phase 3: Metrics & retrospective engine          ✅ COMPLETE
Phase 4: Standup source depth                    ✅ COMPLETE
Phase 5: Sync command                            ⬜ NOT STARTED
```

---

## Phase 0: Activity Logging in Commands — ✅ COMPLETE

Added `db.ActivityLog()` calls (best-effort, non-fatal) to all write commands:

| File | Event Type | Metadata |
|------|-----------|----------|
| `cmd/advance.go` | `"advance"` | `{"from_stage":"X","to_stage":"Y"}` |
| `cmd/revert.go` | `"revert"` | `{"from_stage":"X","to_stage":"Y","reason":"..."}` |
| `cmd/eject.go` | `"eject"` | `{"from_stage":"X","reason":"..."}` |
| `cmd/resume.go` | `"resume"` | `{"to_stage":"X"}` |
| `cmd/decide.go` | `"decide"` / `"decide_resolve"` | `{"number":N}` |

---

## Phase 1: Duration & link_exists Gates — ✅ COMPLETE

### Changes made

**`internal/pipeline/gates.go`**:
- Changed `EvaluateGates` signature to accept `meta *markdown.SpecMeta` (6th param)
- Computes `timeInStage` from `meta.Updated` and populates `expr.Context.Spec.TimeInStage`
- **Duration gate**: Parses `gate.Duration` with `time.ParseDuration()`, compares against `TimeInStage`, shows remaining time on failure
- **link_exists gate**: Extracts URLs from section content via regex, optionally filters by domain type (figma, github, confluence, jira)
- Added helpers: `linkTypeToDomain()`, `formatDuration()`

**Callers updated** (4 sites):
- `cmd/advance.go:90` — passes `meta`
- `cmd/validate.go:56` — passes `meta`
- `cmd/status.go:95` — passes `meta`
- `internal/mcp/handler.go:621` — passes `meta`
- `internal/pipeline/pipeline_test.go` — all 4 calls pass `nil`

---

## Phase 2: Effects Adapter Wiring — ✅ COMPLETE

### Changes made

**New file: `internal/pipeline/effects/adapters.go`**
- `NotifierAdapter` — bridges `CommsAdapter` to the effects `Notifier` interface
- `WebhookerAdapter` — implements `Webhooker` via `net/http` POST
- `LoggerAdapter` — bridges `store.DB` + `markdown.AppendDecision` to the `Logger` interface

**`cmd/advance.go`**:
- Effects `ExecutionContext` now populated with wired `Notifier`, `Webhooker`, `Logger`
- DB opened once before effects, reused for activity logging
- Added `on_exit` effects for departed stage, `on_enter` effects for entered stage
- **Removed legacy comms/PM notification code** (lines 184–203) — pipeline config is now the single source of truth
- Added `runEffects()` helper shared with revert

**`cmd/revert.go`**:
- Same wiring pattern — effects executor with adapters
- Executes `Transitions.Revert.Effects` and `OnEnter` for target stage
- Removed legacy comms notification code
- DB consolidated (one open, reused for effects + activity logging)

---

## Phase 3: Metrics & Retrospective Engine — ✅ COMPLETE

### What's done

**New file: `internal/store/activity.go`** (additions):
- `ActivityCountByType(since time.Time) (map[string]int, error)` — groups events by type
- `ActivityForType(eventType string, since time.Time) ([]ActivityEntry, error)` — filters by type, ordered ASC

**New file: `internal/metrics/metrics.go`**:
- `PipelineMetrics` struct: SpecsCompleted, AvgTimePerStage, ReversionRate, BottleneckStage, SpecsPerStage, TotalAdvances, TotalReversions
- `Compute()` function: derives metrics from activity entries
- `FormatDuration()` helper
- Time-in-stage calculation from consecutive advance events per spec

**New file: `internal/metrics/metrics_test.go`**:
- Tests: empty data, advances+reversions, time-in-stage, bottleneck detection, FormatDuration
- All passing ✅

### Wiring completed

1. **`cmd/metrics.go`** — fully wired:
   - Opens DB, loads activity via `db.ActivitySince()`
   - Scans spec files for current stage distribution via `scanSpecsByStage()`
   - Derives terminal stages from shared `pipeline.TerminalStages()`
   - Calls `metrics.Compute()` and renders formatted output
   - Added `--since` flag (supports `Nd` and Go duration syntax)
   - Added `parseSinceFlag()`, `scanSpecsByStage()` helpers

2. **`cmd/retro.go`** — fully wired:
   - Same metrics computation as `cmd/metrics.go`
   - Shows reversions and ejections detail from activity log
   - `--write` flag writes per-spec journey + cycle context into `## Retrospective` via `markdown.ReplaceSection()`
   - Per-spec metrics via `metrics.ComputeForSpec()` (stages visited, time per stage, reversions, ejections)
   - Cycle context includes specs completed, reversion rate, bottleneck

### Quality fixes applied

- Moved `deriveTerminalStages` to `internal/pipeline/stages.go` as `TerminalStages()` — shared by metrics, retro, standup
- Fixed malformed markdown table in `buildRetroSection` — switched to list format
- Named constant `blockerLookbackDays` replaces magic number in standup
- Documented intentional `openDB` error discard in advance/revert with explanatory comment
- Per-spec retro metrics via `internal/metrics/spec_metrics.go` — each spec gets its own journey data
- Test coverage for all cmd helpers and new engine code

---

## Phase 4: Standup Source Depth — ✅ COMPLETE

### Changes made

**`cmd/standup.go`**:
1. Refactored `printBlockers()` → `collectBlockers()` returning `[]string` — blockers now populate `StandupReport.Blockers`
2. Added PR review requests via `reg.Repo().RequestedReviews(ctx(), userHandle)` — shown in output and added to `today` list
3. Added owned specs in active stages via `collectOwnedSpecs()` — scans spec files, checks stage ownership against pipeline config via `stage.HasOwner(role)`
4. Added mentions via `reg.Comms().FetchMentions(ctx(), since)` — shown in a "Recent mentions" section
5. Moved `buildRegistry` call to function entry — shared between enrichment sources and comms posting

---

## Phase 5: Sync Command — ⬜ NOT STARTED

### New files to create
- `internal/store/sync.go` — `SyncStateGet()`, `SyncStateSet()` methods for the existing `sync_state` table
- `internal/store/sync_test.go`
- `internal/sync/engine.go` — sync engine with outbound/inbound/both flows, conflict detection, section ownership
- `internal/sync/engine_test.go`

### Files to modify
- `cmd/sync.go` — replace stub with real wiring
- `internal/pipeline/effects/adapters.go` — upgrade `SyncerAdapter` to use sync engine

### Key design points
- Outbound: `Docs.PushFull()` + hash each section → `SyncStateSet()`
- Inbound: `Docs.FetchSections()` → compare hashes → apply conflict strategy (warn/abort/force) → respect `<!-- owner: role -->` markers
- Both: inbound first, then outbound
- Hash: `crypto/sha256` on `strings.TrimSpace(content)`

---

## Full File Manifest

### New files (created or to create)
| File | Phase | Status |
|------|-------|--------|
| `internal/pipeline/effects/adapters.go` | 2 | ✅ Created |
| `internal/metrics/metrics.go` | 3 | ✅ Created |
| `internal/metrics/metrics_test.go` | 3 | ✅ Created |
| `internal/metrics/spec_metrics.go` | 3 | ✅ Created |
| `internal/metrics/spec_metrics_test.go` | 3 | ✅ Created |
| `internal/pipeline/stages_test.go` | 3 | ✅ Created |
| `cmd/metrics_helpers_test.go` | 3 | ✅ Created |
| `cmd/retro_helpers_test.go` | 3 | ✅ Created |
| `cmd/standup_helpers_test.go` | 4 | ✅ Created |
| `internal/store/sync.go` | 5 | ⬜ |
| `internal/store/sync_test.go` | 5 | ⬜ |
| `internal/sync/engine.go` | 5 | ⬜ |
| `internal/sync/engine_test.go` | 5 | ⬜ |

### Modified files
| File | Phases | Status |
|------|--------|--------|
| `cmd/advance.go` | 0, 1, 2 | ✅ |
| `cmd/revert.go` | 0, 2 | ✅ |
| `cmd/eject.go` | 0 | ✅ |
| `cmd/resume.go` | 0 | ✅ |
| `cmd/decide.go` | 0 | ✅ |
| `cmd/validate.go` | 1 | ✅ |
| `cmd/status.go` | 1 | ✅ |
| `internal/pipeline/gates.go` | 1 | ✅ |
| `internal/pipeline/pipeline_test.go` | 1 | ✅ |
| `internal/pipeline/stages.go` | 3 | ✅ |
| `internal/mcp/handler.go` | 1 | ✅ |
| `internal/store/activity.go` | 3 | ✅ |
| `cmd/metrics.go` | 3 | ✅ |
| `cmd/retro.go` | 3 | ✅ |
| `cmd/standup.go` | 4 | ✅ |
| `cmd/sync.go` | 5 | ⬜ |

---

## Verification

After each phase, run:
```bash
go build ./...
go test ./...
go vet ./...
```

Current state: `go build ./...` and `go test ./...` both pass cleanly.

### End-to-end verification per phase

| Phase | Manual test |
|-------|-------------|
| 0 | `spec advance SPEC-001` then check `spec.db` activity table has the event |
| 1 | Configure a `duration: "1h"` gate, try `spec validate` on a freshly updated spec → should fail |
| 2 | Configure `notify: "@qa-team"` effect on a stage transition, advance, verify notification fires |
| 3 | `spec metrics` shows real data from activity log |
| 4 | `spec standup` shows PR review requests, owned specs, populated blockers |
| 5 | `spec sync SPEC-001 --direction out` publishes to Confluence |
