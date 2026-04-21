package cmd

import (
	"fmt"
	"strings"

	"github.com/nexl/spec-cli/internal/build"
	"github.com/spf13/cobra"
)

var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Run spec as a standalone MCP server (stdio transport)",
	RunE:  runMCPServer,
}

func init() {
	mcpServerCmd.Flags().String("spec", "", "spec ID to serve context for (defaults to most recent session)")
	rootCmd.AddCommand(mcpServerCmd)
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	specIDFlag, _ := cmd.Flags().GetString("spec")

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
	if specIDFlag != "" {
		specID = strings.ToUpper(specIDFlag)
	} else {
		recent, err := db.SessionMostRecent()
		if err != nil || recent == "" {
			return fmt.Errorf("no active session — use --spec to specify a spec ID")
		}
		specID = recent
	}

	// Load session
	session, err := build.LoadSession(db, specID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("no session found for %s — start a build with 'spec build %s'", specID, specID)
	}

	// Find spec
	specPath, err := resolveLocalSpecPath(specID)
	if err != nil {
		specPath, err = resolveSpecPath(rc, specID)
		if err != nil {
			return err
		}
	}

	// Assemble context
	buildCtx, err := build.AssembleContext(specPath, session, "")
	if err != nil {
		return err
	}

	// Create MCP server
	server := build.NewMCPServer(session, buildCtx, db, specPath)

	// For now, print the available resources and tools (full stdio MCP transport
	// will use the mcp-go SDK in a future iteration)
	fmt.Println("MCP Server ready (spec context provider)")
	fmt.Printf("Spec: %s\n", specID)
	fmt.Println()
	fmt.Println("Resources:")
	for _, r := range server.ListResources() {
		fmt.Printf("  %s — %s\n", r.URI, r.Name)
	}
	fmt.Println()
	fmt.Println("Tools:")
	for _, t := range build.ToolDefinitions() {
		fmt.Printf("  %s — %s\n", t["name"], t["description"])
	}

	// TODO: Full stdio JSON-RPC transport via mcp-go SDK
	// This will be a blocking loop reading JSON-RPC messages from stdin
	// and writing responses to stdout.
	fmt.Println()
	fmt.Println("Standalone MCP server mode is not yet fully implemented.")
	fmt.Println("Use 'spec build' which starts the MCP server automatically.")

	return nil
}
