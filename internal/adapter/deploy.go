package adapter

import "context"

// DeployAdapter manages deployment integration.
type DeployAdapter interface {
	// Trigger initiates a deployment for the given repos to the target environment.
	Trigger(ctx context.Context, repos []string, env string) (*DeployRun, error)
	// Status polls the deployment run for current state.
	Status(ctx context.Context, run *DeployRun) (*DeployStatus, error)
}
