package adapter

import "context"

// AgentAdapter manages coding agent integration.
type AgentAdapter interface {
	// Invoke spawns the agent as a subprocess. The MCP server is already running.
	// contextFile is the fallback consolidated markdown for non-MCP agents.
	// Invoke blocks until the agent exits.
	Invoke(ctx context.Context, contextFile string, workDir string) error
	// SupportsMCP returns true if the agent natively connects to MCP servers.
	SupportsMCP() bool
}
