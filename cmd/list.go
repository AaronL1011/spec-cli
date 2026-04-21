package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nexl/spec-cli/internal/config"
	gitpkg "github.com/nexl/spec-cli/internal/git"
	"github.com/nexl/spec-cli/internal/markdown"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List specs filtered by role queue",
	RunE:  runList,
}

func init() {
	listCmd.Flags().Bool("all", false, "show all specs across all roles and stages")
	listCmd.Flags().String("role", "", "view from another role's perspective")
	listCmd.Flags().Bool("triage", false, "show open triage items")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	showAll, _ := cmd.Flags().GetBool("all")
	roleFilter, _ := cmd.Flags().GetString("role")
	showTriage, _ := cmd.Flags().GetBool("triage")

	rc, err := resolveConfig()
	if err != nil {
		return err
	}

	if showTriage {
		return listTriage(rc)
	}

	if err := requireTeamConfig(rc); err != nil {
		return err
	}

	// Ensure specs repo is fresh
	if _, err := gitpkg.EnsureSpecsRepo(ctx(), &rc.Team.SpecsRepo); err != nil {
		return fmt.Errorf("syncing specs repo: %w", err)
	}

	pipeline := rc.Pipeline()

	// Determine the user's role
	userRole := roleFilter
	if userRole == "" {
		var err error
		userRole, err = requireRole(rc)
		if err != nil {
			return err
		}
	}

	// Read all specs
	specs, err := loadAllSpecs(rc)
	if err != nil {
		return err
	}

	if showAll {
		return listAllByStage(specs, pipeline)
	}

	return listByRole(specs, pipeline, userRole)
}

func listTriage(rc *config.ResolvedConfig) error {
	triageFiles, err := gitpkg.ListTriageFiles(&rc.Team.SpecsRepo)
	if err != nil {
		return err
	}

	if len(triageFiles) == 0 {
		fmt.Println("✓ No open triage items.")
		return nil
	}

	fmt.Println("Open triage items:")
	fmt.Println()
	for _, f := range triageFiles {
		path := gitpkg.TriageFilePath(&rc.Team.SpecsRepo, f)
		meta, err := markdown.ReadTriageMeta(path)
		if err != nil {
			continue
		}
		priorityIcon := priorityIndicator(meta.Priority)
		fmt.Printf("  %s %s  %s  [%s]\n", priorityIcon, meta.ID, meta.Title, meta.Priority)
	}
	return nil
}

type specSummary struct {
	ID     string
	Title  string
	Status string
}

func loadAllSpecs(rc *config.ResolvedConfig) ([]specSummary, error) {
	specFiles, err := gitpkg.ListSpecFiles(&rc.Team.SpecsRepo)
	if err != nil {
		return nil, err
	}

	var specs []specSummary
	for _, f := range specFiles {
		path := filepath.Join(rc.SpecsRepoDir, f)
		meta, err := markdown.ReadMeta(path)
		if err != nil {
			continue
		}
		specs = append(specs, specSummary{
			ID:     meta.ID,
			Title:  meta.Title,
			Status: meta.Status,
		})
	}
	return specs, nil
}

func listByRole(specs []specSummary, pipeline config.PipelineConfig, role string) error {
	var matching []specSummary
	for _, s := range specs {
		stage := pipeline.StageByName(s.Status)
		if stage != nil && stage.OwnerRole == role {
			matching = append(matching, s)
		}
	}

	if len(matching) == 0 {
		fmt.Printf("✓ Nothing awaiting your action. Run 'spec list --all' to see the full pipeline.\n")
		return nil
	}

	fmt.Printf("Specs awaiting %s action:\n\n", role)
	for _, s := range matching {
		fmt.Printf("  %-10s  %-40s  [%s]\n", s.ID, truncate(s.Title, 40), s.Status)
	}
	return nil
}

func listAllByStage(specs []specSummary, pipeline config.PipelineConfig) error {
	if len(specs) == 0 {
		fmt.Println("✓ No specs in the pipeline.")
		return nil
	}

	// Group by stage
	byStage := make(map[string][]specSummary)
	for _, s := range specs {
		byStage[s.Status] = append(byStage[s.Status], s)
	}

	for _, stage := range pipeline.Stages {
		items := byStage[stage.Name]
		if len(items) == 0 {
			continue
		}
		fmt.Printf("─── %s (%s) ───\n", strings.ToUpper(stage.Name), stage.OwnerRole)
		for _, s := range items {
			fmt.Printf("  %-10s  %s\n", s.ID, s.Title)
		}
		fmt.Println()
	}

	// Show blocked separately
	if items := byStage["blocked"]; len(items) > 0 {
		fmt.Printf("─── BLOCKED ───\n")
		for _, s := range items {
			fmt.Printf("  🚫 %-10s  %s\n", s.ID, s.Title)
		}
		fmt.Println()
	}

	fmt.Printf("%d specs in pipeline.\n", len(specs))
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func priorityIndicator(priority string) string {
	switch priority {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "·"
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
