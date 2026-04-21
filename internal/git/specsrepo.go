package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nexl/spec-cli/internal/config"
)

const (
	maxPushRetries = 3
)

// SpecsRepoDir returns the local path for the specs repo clone.
func SpecsRepoDir(cfg *config.SpecsRepoConfig) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".spec", "repos", cfg.Owner, cfg.Repo)
}

// SpecsRepoURL returns the clone URL for the specs repo.
func SpecsRepoURL(cfg *config.SpecsRepoConfig) string {
	switch cfg.Provider {
	case "github":
		return fmt.Sprintf("https://github.com/%s/%s.git", cfg.Owner, cfg.Repo)
	case "gitlab":
		return fmt.Sprintf("https://gitlab.com/%s/%s.git", cfg.Owner, cfg.Repo)
	case "bitbucket":
		return fmt.Sprintf("https://bitbucket.org/%s/%s.git", cfg.Owner, cfg.Repo)
	default:
		return fmt.Sprintf("https://github.com/%s/%s.git", cfg.Owner, cfg.Repo)
	}
}

// EnsureSpecsRepo clones the specs repo if not present, otherwise fetches latest.
func EnsureSpecsRepo(ctx context.Context, cfg *config.SpecsRepoConfig) (string, error) {
	dir := SpecsRepoDir(cfg)

	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		// Clone
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
			return "", fmt.Errorf("creating repos directory: %w", err)
		}
		url := SpecsRepoURL(cfg)
		if err := Clone(ctx, url, dir); err != nil {
			return "", fmt.Errorf("cloning specs repo: %w", err)
		}
		return dir, nil
	}

	// Fetch and reset to latest
	if err := Fetch(ctx, dir); err != nil {
		return dir, fmt.Errorf("fetching specs repo: %w", err)
	}
	ref := fmt.Sprintf("origin/%s", cfg.Branch)
	if err := ResetHard(ctx, dir, ref); err != nil {
		return dir, fmt.Errorf("resetting specs repo: %w", err)
	}

	return dir, nil
}

// WithSpecsRepo fetches the latest, calls the mutator function, then commits and pushes.
// If the push fails due to a conflict, it retries up to maxPushRetries times.
func WithSpecsRepo(ctx context.Context, cfg *config.SpecsRepoConfig, mutate func(repoPath string) (commitMsg string, err error)) error {
	dir := SpecsRepoDir(cfg)

	for attempt := 0; attempt <= maxPushRetries; attempt++ {
		// Fetch latest
		if err := Fetch(ctx, dir); err != nil {
			return fmt.Errorf("fetching specs repo: %w", err)
		}
		ref := fmt.Sprintf("origin/%s", cfg.Branch)
		if err := ResetHard(ctx, dir, ref); err != nil {
			return fmt.Errorf("resetting specs repo: %w", err)
		}

		// Apply mutation
		commitMsg, err := mutate(dir)
		if err != nil {
			return fmt.Errorf("mutation failed: %w", err)
		}

		// Check if there are changes to commit
		hasChanges, err := HasChanges(ctx, dir)
		if err != nil {
			return fmt.Errorf("checking changes: %w", err)
		}
		if !hasChanges {
			return nil // Nothing to do
		}

		// Commit
		if err := Commit(ctx, dir, commitMsg); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		// Push
		if err := Push(ctx, dir, cfg.Branch); err != nil {
			if attempt < maxPushRetries {
				// Retry: concurrent push may have advanced the ref
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			return fmt.Errorf("push failed after %d retries — another user may have modified the specs repo: %w", maxPushRetries, err)
		}

		return nil
	}

	return fmt.Errorf("push failed after %d retries", maxPushRetries)
}

// ReadSpecFile reads a spec file from the specs repo.
func ReadSpecFile(cfg *config.SpecsRepoConfig, filename string) ([]byte, error) {
	dir := SpecsRepoDir(cfg)
	path := filepath.Join(dir, filename)
	return os.ReadFile(path)
}

// ListSpecFiles returns all spec files in the specs repo root.
func ListSpecFiles(cfg *config.SpecsRepoConfig) ([]string, error) {
	dir := SpecsRepoDir(cfg)
	return listMarkdownFiles(dir)
}

// ListTriageFiles returns all triage files in the triage/ directory.
func ListTriageFiles(cfg *config.SpecsRepoConfig) ([]string, error) {
	dir := SpecsRepoDir(cfg)
	triageDir := filepath.Join(dir, "triage")
	if _, err := os.Stat(triageDir); os.IsNotExist(err) {
		return nil, nil
	}
	return listMarkdownFiles(triageDir)
}

// ListArchiveFiles returns all archived spec files.
func ListArchiveFiles(cfg *config.SpecsRepoConfig, archiveDir string) ([]string, error) {
	dir := SpecsRepoDir(cfg)
	archivePath := filepath.Join(dir, archiveDir)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return nil, nil
	}
	return listMarkdownFiles(archivePath)
}

func listMarkdownFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

// SpecFilePath returns the absolute path to a spec file in the specs repo.
func SpecFilePath(cfg *config.SpecsRepoConfig, filename string) string {
	return filepath.Join(SpecsRepoDir(cfg), filename)
}

// TriageFilePath returns the absolute path to a triage file.
func TriageFilePath(cfg *config.SpecsRepoConfig, filename string) string {
	return filepath.Join(SpecsRepoDir(cfg), "triage", filename)
}
