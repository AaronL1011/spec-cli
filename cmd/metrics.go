package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show quantitative pipeline health metrics",
	RunE:  runMetrics,
}

func init() {
	metricsCmd.Flags().String("cycle", "", "metrics for a specific cycle")
	rootCmd.AddCommand(metricsCmd)
}

func runMetrics(cmd *cobra.Command, args []string) error {
	rc, err := resolveConfig()
	if err != nil {
		return err
	}

	cycle, _ := cmd.Flags().GetString("cycle")
	if cycle == "" {
		cycle = rc.CycleLabel()
	}

	fmt.Printf("Pipeline metrics — %s\n", cycle)
	fmt.Println("────────────────────────────────────────────────")

	// TODO: Real metrics from activity log
	fmt.Println()
	fmt.Println("Full metrics engine ships in v0.4")

	return nil
}
