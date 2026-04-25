package metrics

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/aaronl1011/spec-cli/internal/store"
)

// PipelineMetrics holds computed metrics for a cycle or time window.
type PipelineMetrics struct {
	SpecsCompleted  int
	AvgTimePerStage map[string]time.Duration
	ReversionRate   float64
	BottleneckStage string
	SpecsPerStage   map[string]int
	TotalAdvances   int
	TotalReversions int
}

type advanceMeta struct {
	FromStage string `json:"from_stage"`
	ToStage   string `json:"to_stage"`
}

// Compute calculates pipeline metrics from activity entries and current spec distribution.
func Compute(entries []store.ActivityEntry, specsByStage map[string]int, stageNames []string, terminalStages []string) *PipelineMetrics {
	m := &PipelineMetrics{
		AvgTimePerStage: make(map[string]time.Duration),
		SpecsPerStage:   specsByStage,
	}

	if len(entries) == 0 {
		return m
	}

	// Count advances and reversions
	var advances []store.ActivityEntry
	for _, e := range entries {
		switch e.EventType {
		case "advance":
			m.TotalAdvances++
			advances = append(advances, e)
		case "revert":
			m.TotalReversions++
		}
	}

	if m.TotalAdvances > 0 {
		m.ReversionRate = float64(m.TotalReversions) / float64(m.TotalAdvances)
	}

	// Compute time-in-stage from consecutive advance events per spec
	stageDurations := computeStageDurations(advances)
	for stage, durations := range stageDurations {
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		m.AvgTimePerStage[stage] = total / time.Duration(len(durations))
	}

	// Find bottleneck (longest avg dwell among configured stages)
	var maxDwell time.Duration
	for _, name := range stageNames {
		if avg, ok := m.AvgTimePerStage[name]; ok && avg > maxDwell {
			maxDwell = avg
			m.BottleneckStage = name
		}
	}

	// Count completed specs
	terminalSet := make(map[string]bool, len(terminalStages))
	for _, s := range terminalStages {
		terminalSet[s] = true
	}

	lastStagePerSpec := make(map[string]string)
	for _, e := range advances {
		var meta advanceMeta
		if json.Unmarshal([]byte(e.Metadata), &meta) == nil && meta.ToStage != "" {
			lastStagePerSpec[e.SpecID] = meta.ToStage
		}
	}
	for _, toStage := range lastStagePerSpec {
		if terminalSet[toStage] {
			m.SpecsCompleted++
		}
	}

	return m
}

func computeStageDurations(advances []store.ActivityEntry) map[string][]time.Duration {
	// Group by spec, sorted by time (already ASC from query)
	bySpec := make(map[string][]store.ActivityEntry)
	for _, e := range advances {
		bySpec[e.SpecID] = append(bySpec[e.SpecID], e)
	}

	durations := make(map[string][]time.Duration)
	for _, specAdvances := range bySpec {
		sort.Slice(specAdvances, func(i, j int) bool {
			return specAdvances[i].CreatedAt.Before(specAdvances[j].CreatedAt)
		})

		for i := 0; i < len(specAdvances)-1; i++ {
			var meta advanceMeta
			if json.Unmarshal([]byte(specAdvances[i].Metadata), &meta) != nil || meta.FromStage == "" {
				continue
			}
			dwell := specAdvances[i+1].CreatedAt.Sub(specAdvances[i].CreatedAt)
			durations[meta.FromStage] = append(durations[meta.FromStage], dwell)
		}
	}
	return durations
}

// FormatDuration renders a duration in a human-friendly way.
func FormatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	if hours < 24 {
		mins := int(d.Minutes()) % 60
		if mins == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	days := hours / 24
	remainHours := hours % 24
	if remainHours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, remainHours)
}
