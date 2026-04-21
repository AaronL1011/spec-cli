package cmd

import (
	"fmt"
	"strings"

	"github.com/nexl/spec-cli/internal/markdown"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy <id>",
	Short: "Trigger deployment via CI/CD adapter",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().String("env", "staging", "target environment")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	specID := strings.ToUpper(args[0])
	env, _ := cmd.Flags().GetString("env")

	rc, err := resolveConfig()
	if err != nil {
		return err
	}

	if !rc.HasIntegration("deploy") {
		return fmt.Errorf("deploy integration not configured — add 'integrations.deploy' to spec.config.yaml")
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
		return fmt.Errorf("no repos listed in %s — add 'repos:' to the frontmatter", specID)
	}

	reg := buildRegistry(rc)

	fmt.Printf("Deploying %s to %s...\n", specID, env)
	for _, repo := range meta.Repos {
		fmt.Printf("  Triggering %s...\n", repo)
	}

	run, err := reg.Deploy().Trigger(ctx(), meta.Repos, env)
	if err != nil {
		return fmt.Errorf("triggering deployment: %w", err)
	}

	if run != nil {
		fmt.Printf("✓ Deployment triggered: %s\n", run.URL)
	} else {
		fmt.Println("✓ Deployment triggered.")
	}

	return nil
}
