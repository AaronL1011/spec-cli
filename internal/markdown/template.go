package markdown

import (
	"fmt"
	"time"
)

// ScaffoldSpec generates a new SPEC.md from the template.
func ScaffoldSpec(id, title, author, cycle, source string) string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`---
id: %s
title: %s
status: draft
version: 0.1.0
author: %s
cycle: %s
epic_key: ""
repos: []
revert_count: 0
source: %s
created: %s
updated: %s
---

# %s - %s

## Decision Log
| # | Question / Decision | Options Considered | Decision Made | Rationale | Decided By | Date |
|---|---|---|---|---|---|---|

## 1. Problem Statement           <!-- owner: pm -->

## 2. Goals & Non-Goals           <!-- owner: pm -->

## 3. User Stories                <!-- owner: pm -->

## 4. Proposed Solution           <!-- owner: pm -->

### 4.1 Concept Overview

### 4.2 Architecture / Approach

## 5. Design Inputs               <!-- owner: designer -->

## 6. Acceptance Criteria         <!-- owner: qa -->

## 7. Technical Implementation    <!-- owner: engineer -->

### 7.1 Architecture Notes

### 7.2 Dependencies & Risks

### 7.3 PR Stack Plan

## 8. Escape Hatch Log            <!-- auto: spec eject -->

## 9. QA Validation Notes         <!-- owner: qa -->

## 10. Deployment Notes           <!-- owner: engineer -->

## 11. Retrospective              <!-- auto: spec retro -->
`, id, title, author, cycle, source, date, date, id, title)
}

// ScaffoldTriage generates a new TRIAGE.md from the template.
func ScaffoldTriage(id, title, priority, source, sourceRef, reportedBy string) string {
	if priority == "" {
		priority = "medium"
	}
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`---
id: %s
title: %s
status: triage
priority: %s
source: %s
source_ref: %s
reported_by: %s
created: %s
---

# %s - %s

## Context

## Notes
`, id, title, priority, source, sourceRef, reportedBy, date, id, title)
}

// NextSpecID scans existing spec filenames and returns the next sequential ID.
// Files should be like SPEC-001.md, SPEC-002.md, etc.
func NextSpecID(existingFiles []string) string {
	maxNum := 0
	for _, f := range existingFiles {
		var num int
		if _, err := fmt.Sscanf(f, "SPEC-%d.md", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return fmt.Sprintf("SPEC-%03d", maxNum+1)
}

// NextTriageID scans existing triage filenames and returns the next sequential ID.
func NextTriageID(existingFiles []string) string {
	maxNum := 0
	for _, f := range existingFiles {
		var num int
		if _, err := fmt.Sscanf(f, "TRIAGE-%d.md", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return fmt.Sprintf("TRIAGE-%03d", maxNum+1)
}
