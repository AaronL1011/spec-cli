package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTeamConfig(t *testing.T) {
	content := `
version: "1"
team:
  name: "Test Team"
  cycle_label: "Cycle 1"
specs_repo:
  provider: github
  owner: test-org
  repo: specs
  branch: main
integrations:
  comms:
    provider: slack
  pm:
    provider: jira
    base_url: https://jira.example.com
  docs:
    provider: none
  repo:
    provider: github
  agent:
    provider: claude-code
  ai:
    provider: anthropic
    model: claude-sonnet-4-20250514
pipeline:
  stages:
    - name: triage
      owner_role: pm
    - name: draft
      owner_role: pm
    - name: build
      owner_role: engineer
      gates:
        - section_complete: acceptance_criteria
    - name: done
      owner_role: tl
      optional: true
      auto_archive: true
`

	dir := t.TempDir()
	path := filepath.Join(dir, "spec.config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadTeamConfig(path)
	if err != nil {
		t.Fatalf("LoadTeamConfig: %v", err)
	}

	if cfg.Team.Name != "Test Team" {
		t.Errorf("team.name = %q, want %q", cfg.Team.Name, "Test Team")
	}
	if cfg.SpecsRepo.Provider != "github" {
		t.Errorf("specs_repo.provider = %q, want %q", cfg.SpecsRepo.Provider, "github")
	}
	if cfg.Integrations.PM.Provider != "jira" {
		t.Errorf("integrations.pm.provider = %q, want %q", cfg.Integrations.PM.Provider, "jira")
	}
	if cfg.Integrations.AI.Provider != "anthropic" {
		t.Errorf("integrations.ai.provider = %q, want %q", cfg.Integrations.AI.Provider, "anthropic")
	}
	if len(cfg.Pipeline.Stages) != 4 {
		t.Fatalf("pipeline.stages length = %d, want 4", len(cfg.Pipeline.Stages))
	}
	if cfg.Pipeline.Stages[2].Name != "build" {
		t.Errorf("stages[2].name = %q, want %q", cfg.Pipeline.Stages[2].Name, "build")
	}
	if len(cfg.Pipeline.Stages[2].Gates) != 1 {
		t.Fatalf("stages[2].gates length = %d, want 1", len(cfg.Pipeline.Stages[2].Gates))
	}
	if cfg.Pipeline.Stages[2].Gates[0].Type() != "section_complete" {
		t.Errorf("gate type = %q, want %q", cfg.Pipeline.Stages[2].Gates[0].Type(), "section_complete")
	}
	if !cfg.Pipeline.Stages[3].Optional {
		t.Error("stages[3].optional = false, want true")
	}
}

func TestEnvVarInterpolation(t *testing.T) {
	t.Setenv("TEST_TOKEN", "secret123")
	content := `
version: "1"
team:
  name: "Test"
specs_repo:
  provider: github
  owner: org
  repo: specs
  token: ${TEST_TOKEN}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadTeamConfig(path)
	if err != nil {
		t.Fatalf("LoadTeamConfig: %v", err)
	}

	if cfg.SpecsRepo.Token != "secret123" {
		t.Errorf("token = %q, want %q", cfg.SpecsRepo.Token, "secret123")
	}
}

func TestPipelineStageIndex(t *testing.T) {
	p := DefaultPipeline()

	tests := []struct {
		name string
		want int
	}{
		{"triage", 0},
		{"draft", 1},
		{"build", 6},
		{"done", 9},
		{"nonexistent", -1},
	}

	for _, tt := range tests {
		got := p.StageIndex(tt.name)
		if got != tt.want {
			t.Errorf("StageIndex(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestPipelineTransitions(t *testing.T) {
	p := DefaultPipeline()

	if !p.IsValidTransition("draft", "tl-review") {
		t.Error("draft → tl-review should be valid")
	}
	if p.IsValidTransition("build", "draft") {
		t.Error("build → draft should be invalid (backward)")
	}
	if !p.IsValidReversion("build", "draft") {
		t.Error("build → draft reversion should be valid")
	}
	if p.IsValidReversion("draft", "build") {
		t.Error("draft → build reversion should be invalid (forward)")
	}
}

func TestFindTeamConfigPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "spec.config.yaml")
	if err := os.WriteFile(cfgPath, []byte("version: '1'"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory and search from there
	subDir := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := FindTeamConfigPath(subDir)
	if err != nil {
		t.Fatalf("FindTeamConfigPath: %v", err)
	}
	if found != cfgPath {
		t.Errorf("found = %q, want %q", found, cfgPath)
	}
}

func TestDefaultPipelineHasAllStages(t *testing.T) {
	p := DefaultPipeline()
	expected := []string{
		"triage", "draft", "tl-review", "design", "qa-expectations",
		"engineering", "build", "pr-review", "qa-validation", "done",
		"deploying", "monitoring", "closed",
	}

	names := p.StageNames()
	if len(names) != len(expected) {
		t.Fatalf("stage count = %d, want %d", len(names), len(expected))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("stage[%d] = %q, want %q", i, name, expected[i])
		}
	}
}
