package cmd

import (
	"fmt"
	"time"

	"github.com/nexl/spec-cli/internal/store"
	"github.com/spf13/cobra"
)

var standupCmd = &cobra.Command{
	Use:   "standup",
	Short: "Auto-generate standup from actual activity",
	RunE:  runStandup,
}

func init() {
	rootCmd.AddCommand(standupCmd)
}

func runStandup(cmd *cobra.Command, args []string) error {
	rc, err := resolveConfig()
	if err != nil {
		return err
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Get activity from last 24h
	since := time.Now().Add(-24 * time.Hour)
	entries, err := db.ActivitySince(since)
	if err != nil {
		return err
	}

	userName := rc.UserName()
	date := time.Now().Format("2006-01-02")

	fmt.Printf("Your standup — %s — %s\n", userName, date)
	fmt.Println("────────────────────────────────────────────────")

	// Yesterday
	fmt.Println("Yesterday:")
	if len(entries) == 0 {
		fmt.Println("  (no tracked activity)")
	} else {
		for _, e := range entries {
			fmt.Printf("  • %s: %s\n", e.SpecID, e.Summary)
		}
	}

	// Today (from active session)
	fmt.Println("\nToday:")
	recent, _ := db.SessionMostRecent()
	if recent != "" {
		session, _ := db.SessionGet(recent)
		if session != "" {
			fmt.Printf("  • Continue %s\n", recent)
		}
	} else {
		fmt.Println("  (run 'spec do' to start)")
	}

	// Blockers (from blocked specs)
	fmt.Println("\nBlockers:")
	printBlockers(db)

	// Post option
	if rc.HasIntegration("comms") {
		autoPost := false
		if rc.User != nil {
			autoPost = rc.User.Preferences.StandupAutoPost
		}
		if autoPost {
			fmt.Println("\nPosting to standup channel...")
			// reg := buildRegistry(rc)
			// _ = reg.Comms().PostStandup(ctx(), ...)
		} else {
			fmt.Println("\nPost to standup channel? [y/N]")
		}
	}

	return nil
}

func printBlockers(db *store.DB) {
	// Check for blocked specs from activity log
	entries, err := db.ActivitySince(time.Now().Add(-7 * 24 * time.Hour))
	if err != nil {
		fmt.Println("  (none)")
		return
	}

	hasBlocker := false
	for _, e := range entries {
		if e.EventType == "eject" {
			fmt.Printf("  • %s: %s\n", e.SpecID, e.Summary)
			hasBlocker = true
		}
	}
	if !hasBlocker {
		fmt.Println("  (none)")
	}
}
