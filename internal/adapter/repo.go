package adapter

import "context"

// RepoAdapter manages code repository integration.
type RepoAdapter interface {
	// ListPRs returns open PRs matching a spec's branch pattern across its repos.
	ListPRs(ctx context.Context, repos []string, specID string) ([]PullRequest, error)
	// PRStatus returns the review/CI status of a specific PR.
	PRStatus(ctx context.Context, repo string, prNumber int) (*PRDetail, error)
	// SetPRDescription updates a PR's description.
	SetPRDescription(ctx context.Context, repo string, prNumber int, body string) error
	// RequestedReviews returns PRs where the current user is a requested reviewer.
	RequestedReviews(ctx context.Context, user string) ([]PullRequest, error)
}
