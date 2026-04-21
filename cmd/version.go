package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the spec CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("spec %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
