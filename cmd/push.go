package cmd

import (
	"context"
	"fmt"
	"strings"

	gitpkg "github.com/nexl/spec-cli/internal/git"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <id>",
	Short: "Commit and push local spec edits to the specs repo",
	Long:  "Commits any uncommitted local changes to the spec and pushes them to the remote specs repo. Run this after 'spec edit' to make your edits visible to the rest of the team.",
	Args:  cobra.ExactArgs(1),
	RunE:  runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
	specID := strings.ToUpper(args[0])

	rc, err := resolveConfig()
	if err != nil {
		return err
	}
	if err := requireTeamConfig(rc); err != nil {
		return err
	}

	pushed, err := gitpkg.PushLocalEdits(
		context.Background(),
		&rc.Team.SpecsRepo,
		fmt.Sprintf("feat: update %s", specID),
	)
	if err != nil {
		return fmt.Errorf("pushing %s: %w", specID, err)
	}

	if !pushed {
		fmt.Printf("Nothing to push — no local changes found for %s\n", specID)
		return nil
	}

	fmt.Printf("✓ %s pushed to specs repo\n", specID)
	return nil
}
