package pipeline

import (
	"fmt"
	"strings"

	"github.com/nexl/spec-cli/internal/config"
	"github.com/nexl/spec-cli/internal/markdown"
)

// GateResult represents the result of a gate check.
type GateResult struct {
	Gate   string
	Passed bool
	Reason string
}

// EvaluateGates checks all gates for the current stage.
func EvaluateGates(pipeline config.PipelineConfig, currentStage string, sections []markdown.Section, hasPRStack bool, prsApproved bool) []GateResult {
	stage := pipeline.StageByName(currentStage)
	if stage == nil {
		return nil
	}

	var results []GateResult
	for _, gate := range stage.Gates {
		result := evaluateGate(gate, sections, hasPRStack, prsApproved)
		results = append(results, result)
	}
	return results
}

// AllGatesPassed returns true if all gates passed.
func AllGatesPassed(results []GateResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}

// FailedGates returns only the gates that did not pass.
func FailedGates(results []GateResult) []GateResult {
	var failed []GateResult
	for _, r := range results {
		if !r.Passed {
			failed = append(failed, r)
		}
	}
	return failed
}

func evaluateGate(gate config.GateConfig, sections []markdown.Section, hasPRStack bool, prsApproved bool) GateResult {
	gateType := gate.Type()
	gateValue := gate.Value()

	switch gateType {
	case "section_complete":
		slug := gateValue
		if markdown.IsSectionNonEmpty(sections, slug) {
			return GateResult{Gate: fmt.Sprintf("section_complete: %s", slug), Passed: true}
		}
		return GateResult{
			Gate:   fmt.Sprintf("section_complete: %s", slug),
			Passed: false,
			Reason: fmt.Sprintf("section %q is empty — it must have content before advancing", humanizeSlug(slug)),
		}

	case "pr_stack_exists":
		if hasPRStack {
			return GateResult{Gate: "pr_stack_exists", Passed: true}
		}
		return GateResult{
			Gate:   "pr_stack_exists",
			Passed: false,
			Reason: "PR stack plan (§7.3) is required — add the PR stack with 'spec edit' or 'spec draft --pr-stack'",
		}

	case "prs_approved":
		if prsApproved {
			return GateResult{Gate: "prs_approved", Passed: true}
		}
		return GateResult{
			Gate:   "prs_approved",
			Passed: false,
			Reason: "all PRs must be approved before advancing to QA validation",
		}

	case "duration":
		// Duration gates are checked elsewhere (requires timestamp)
		// For now, pass them in validate mode
		return GateResult{Gate: fmt.Sprintf("duration: %s", gateValue), Passed: true}

	default:
		return GateResult{
			Gate:   gateType,
			Passed: true,
			Reason: fmt.Sprintf("unknown gate type %q — skipping", gateType),
		}
	}
}

func humanizeSlug(slug string) string {
	return strings.ReplaceAll(slug, "_", " ")
}
