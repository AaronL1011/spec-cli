package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var retroCmd = &cobra.Command{
	Use:   "retro",
	Short: "Auto-populate retrospective with cycle metrics",
	RunE:  runRetro,
}

func init() {
	retroCmd.Flags().String("cycle", "", "retrospect a specific cycle")
	rootCmd.AddCommand(retroCmd)
}

func runRetro(cmd *cobra.Command, args []string) error {
	rc, err := resolveConfig()
	if err != nil {
		return err
	}

	cycle, _ := cmd.Flags().GetString("cycle")
	if cycle == "" {
		cycle = rc.CycleLabel()
	}

	fmt.Printf("Retrospective — %s\n", cycle)
	fmt.Println("────────────────────────────────────────────────")
	fmt.Println()

	// TODO: Aggregate real metrics from activity log and spec transitions
	fmt.Println("Metrics:")
	fmt.Println("  Specs completed:     (calculating...)")
	fmt.Println("  Avg time per stage:  (calculating...)")
	fmt.Println("  Reversion rate:      (calculating...)")
	fmt.Println("  Bottleneck stage:    (calculating...)")
	fmt.Println()
	fmt.Println("Full metrics engine ships in v0.4")

	return nil
}
