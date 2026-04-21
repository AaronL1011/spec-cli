package cmd

import (
	"fmt"
	"strings"

	"github.com/nexl/spec-cli/internal/adapter"
	"github.com/nexl/spec-cli/internal/markdown"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review <id>",
	Short: "Post structured review request with all stacked PRs",
	Args:  cobra.ExactArgs(1),
	RunE:  runReview,
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}

func runReview(cmd *cobra.Command, args []string) error {
	specID := strings.ToUpper(args[0])

	rc, err := resolveConfig()
	if err != nil {
		return err
	}

	path, err := resolveSpecPath(rc, specID)
	if err != nil {
		return err
	}

	meta, err := markdown.ReadMeta(path)
	if err != nil {
		return err
	}

	if len(meta.Repos) == 0 {
		return fmt.Errorf("no repos listed in %s frontmatter — add 'repos:' to the spec", specID)
	}

	reg := buildRegistry(rc)

	// List PRs from all repos
	prs, err := reg.Repo().ListPRs(ctx(), meta.Repos, specID)
	if err != nil {
		return fmt.Errorf("listing PRs: %w", err)
	}

	if len(prs) == 0 {
		fmt.Printf("No open PRs found for %s across %s\n", specID, strings.Join(meta.Repos, ", "))
		return nil
	}

	// Display PRs in order
	fmt.Printf("Review request for %s — %s\n\n", specID, meta.Title)
	for i, pr := range prs {
		fmt.Printf("  %d. PR #%d — %s (%s)\n", i+1, pr.Number, pr.Title, pr.Repo)
		if pr.URL != "" {
			fmt.Printf("     %s\n", pr.URL)
		}
	}

	// Post to comms — non-fatal, warn on failure
	if rc.HasIntegration("comms") {
		msg := fmt.Sprintf("[%s] Review requested — %s\n", specID, meta.Title)
		for _, pr := range prs {
			msg += fmt.Sprintf("  • PR #%d: %s (%s)\n", pr.Number, pr.Title, pr.Repo)
		}
		if err := reg.Comms().Notify(ctx(), adapter.Notification{
			SpecID:  specID,
			Title:   meta.Title,
			Message: msg,
		}); err != nil {
			warnf("could not send notification: %v", err)
		} else {
			fmt.Println("\n✓ Review request posted to comms.")
		}
	}

	return nil
}
