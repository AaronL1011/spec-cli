package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nexl/spec-cli/internal/build"
	gitpkg "github.com/nexl/spec-cli/internal/git"
	"github.com/spf13/cobra"
)

var doCmd = &cobra.Command{
	Use:   "do [id]",
	Short: "Resume work — picks up where you left off",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDo,
}

func init() {
	rootCmd.AddCommand(doCmd)
}

func runDo(cmd *cobra.Command, args []string) error {
	rc, err := resolveConfig()
	if err != nil {
		return err
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	var specID string

	if len(args) > 0 {
		specID = strings.ToUpper(args[0])
	} else {
		// Try to detect from current branch
		workDir, _ := os.Getwd()
		specID = gitpkg.DetectSpecFromBranch(ctx(), workDir)

		// Try most recent session
		if specID == "" {
			recent, err := db.SessionMostRecent()
			if err == nil && recent != "" {
				specID = recent
			}
		}

		if specID == "" {
			return fmt.Errorf("no active build session found — start one with 'spec build <id>'")
		}
	}

	// Find the spec
	specPath, err := resolveLocalSpecPath(specID)
	if err != nil {
		specPath, err = resolveSpecPath(rc, specID)
		if err != nil {
			return fmt.Errorf("%s not found — run 'spec pull %s' to fetch it", specID, specID)
		}
	}

	reg := buildRegistry(rc)
	engine := build.NewEngine(db, reg.Agent())

	workDir, _ := os.Getwd()
	return engine.StartOrResume(ctx(), specID, specPath, workDir)
}
