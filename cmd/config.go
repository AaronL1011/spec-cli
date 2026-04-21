package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nexl/spec-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage spec configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive wizard for configuration setup",
	RunE:  runConfigInit,
}

var configTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Validate all configured integrations",
	RunE:  runConfigTest,
}

func init() {
	configInitCmd.Flags().Bool("user", false, "initialise personal user config (~/.spec/config.yaml)")
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configTestCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	isUser, _ := cmd.Flags().GetBool("user")

	if isUser {
		return runUserConfigInit()
	}
	return runTeamConfigInit()
}

func runUserConfigInit() error {
	reader := bufio.NewReader(os.Stdin)
	cfg := &config.UserConfig{}
	aiDrafts := true
	cfg.Preferences.AIDrafts = &aiDrafts

	fmt.Println("Setting up your personal spec identity (~/.spec/config.yaml)")
	fmt.Println()

	// Name
	fmt.Print("Your name: ")
	name, _ := reader.ReadString('\n')
	cfg.User.Name = strings.TrimSpace(name)

	// Role
	fmt.Printf("Your role (%s): ", strings.Join(config.ValidRoles(), " | "))
	role, _ := reader.ReadString('\n')
	role = strings.TrimSpace(strings.ToLower(role))
	if !config.IsValidRole(role) {
		return fmt.Errorf("invalid role %q — must be one of: %s", role, strings.Join(config.ValidRoles(), ", "))
	}
	cfg.User.OwnerRole = role

	// Handle
	fmt.Print("Your comms handle (e.g., @aaron or aaron@org.com): ")
	handle, _ := reader.ReadString('\n')
	cfg.User.Handle = strings.TrimSpace(handle)

	// Editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	fmt.Printf("Preferred editor [%s]: ", editor)
	editorInput, _ := reader.ReadString('\n')
	editorInput = strings.TrimSpace(editorInput)
	if editorInput != "" {
		editor = editorInput
	}
	cfg.Preferences.Editor = editor

	path := config.UserConfigPath()
	if err := config.WriteUserConfig(path, cfg); err != nil {
		return err
	}

	fmt.Printf("\n✓ User config written to %s\n", path)
	return nil
}

func runTeamConfigInit() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Setting up team spec configuration (spec.config.yaml)")
	fmt.Println()

	// Team name
	fmt.Print("Team name: ")
	teamName, _ := reader.ReadString('\n')
	teamName = strings.TrimSpace(teamName)

	// Cycle label
	fmt.Print("Current cycle label (e.g., Cycle 7): ")
	cycleLabel, _ := reader.ReadString('\n')
	cycleLabel = strings.TrimSpace(cycleLabel)

	// Specs repo provider
	fmt.Print("Specs repo provider (github | gitlab | bitbucket): ")
	repoProvider, _ := reader.ReadString('\n')
	repoProvider = strings.TrimSpace(strings.ToLower(repoProvider))
	if repoProvider == "" {
		repoProvider = "github"
	}

	// Specs repo owner
	fmt.Print("Specs repo owner/org: ")
	repoOwner, _ := reader.ReadString('\n')
	repoOwner = strings.TrimSpace(repoOwner)

	// Specs repo name
	fmt.Print("Specs repo name [specs]: ")
	repoName, _ := reader.ReadString('\n')
	repoName = strings.TrimSpace(repoName)
	if repoName == "" {
		repoName = "specs"
	}

	content := fmt.Sprintf(`version: "1"

team:
  name: %q
  cycle_label: %q

specs_repo:
  provider: %s
  owner: %s
  repo: %s
  branch: main
  token: ${GITHUB_TOKEN}

integrations:
  comms:
    provider: none
  pm:
    provider: none
  docs:
    provider: none
  repo:
    provider: %s
    owner: %s
    token: ${GITHUB_TOKEN}
  agent:
    provider: none
  ai:
    provider: none
  design:
    provider: none
  deploy:
    provider: none

sync:
  outbound_on_advance: true
  conflict_strategy: warn

archive:
  directory: archive

dashboard:
  stale_threshold: 48h
  refresh_ttl: 300

pipeline:
  stages:
    - name: triage
      owner_role: pm
    - name: draft
      owner_role: pm
    - name: tl-review
      owner_role: tl
      gates:
        - section_complete: problem_statement
    - name: design
      owner_role: designer
      gates:
        - section_complete: user_stories
    - name: qa-expectations
      owner_role: qa
      gates:
        - section_complete: design_inputs
    - name: engineering
      owner_role: engineer
      gates:
        - section_complete: acceptance_criteria
    - name: build
      owner_role: engineer
    - name: pr-review
      owner_role: engineer
      gates:
        - pr_stack_exists: true
    - name: qa-validation
      owner_role: qa
      gates:
        - prs_approved: true
    - name: done
      owner_role: tl
    - name: deploying
      owner_role: engineer
      optional: true
    - name: monitoring
      owner_role: engineer
      optional: true
      gates:
        - duration: 24h
    - name: closed
      owner_role: tl
      optional: true
      auto_archive: true
`, teamName, cycleLabel, repoProvider, repoOwner, repoName, repoProvider, repoOwner)

	path := "spec.config.yaml"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing team config: %w", err)
	}

	fmt.Printf("\n✓ Team config written to %s\n", path)
	fmt.Println("  Edit the integrations section to connect your tools.")
	fmt.Println("  Commit this file to your specs repo.")
	return nil
}

func runConfigTest(cmd *cobra.Command, args []string) error {
	rc, err := config.Resolve()
	if err != nil {
		return err
	}

	fmt.Println("Configuration test results:")
	fmt.Println()

	// User config
	if rc.User != nil && rc.User.User.OwnerRole != "" {
		fmt.Printf("  ✓ User config: %s (role: %s)\n", rc.UserConfigPath, rc.User.User.OwnerRole)
	} else {
		fmt.Printf("  ✗ User config: not configured — run 'spec config init --user'\n")
	}

	// Team config
	if rc.Team != nil {
		fmt.Printf("  ✓ Team config: %s (team: %s)\n", rc.TeamConfigPath, rc.Team.Team.Name)
	} else {
		fmt.Printf("  ✗ Team config: not found — run 'spec config init'\n")
	}

	if rc.Team == nil {
		return nil
	}

	// Check integrations
	categories := []struct {
		name     string
		category string
	}{
		{"Comms", "comms"},
		{"PM", "pm"},
		{"Docs", "docs"},
		{"Repo", "repo"},
		{"Agent", "agent"},
		{"AI", "ai"},
		{"Design", "design"},
		{"Deploy", "deploy"},
	}

	fmt.Println()
	fmt.Println("  Integrations:")
	for _, cat := range categories {
		if rc.HasIntegration(cat.category) {
			fmt.Printf("    ✓ %s: configured\n", cat.name)
		} else {
			fmt.Printf("    · %s: not configured\n", cat.name)
		}
	}

	return nil
}
