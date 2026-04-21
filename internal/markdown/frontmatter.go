// Package markdown provides parsing and mutation for SPEC.md files.
// It operates on line-level patterns, not a full AST — sufficient for the
// structured SPEC.md format.
package markdown

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SpecMeta represents the YAML frontmatter of a SPEC.md file.
type SpecMeta struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Status      string   `yaml:"status"`
	Version     string   `yaml:"version"`
	Author      string   `yaml:"author"`
	Cycle       string   `yaml:"cycle"`
	EpicKey     string   `yaml:"epic_key,omitempty"`
	Repos       []string `yaml:"repos,omitempty"`
	RevertCount int      `yaml:"revert_count"`
	Source      string   `yaml:"source,omitempty"`
	Created     string   `yaml:"created"`
	Updated     string   `yaml:"updated"`
}

// TriageMeta represents the YAML frontmatter of a TRIAGE.md file.
type TriageMeta struct {
	ID         string `yaml:"id"`
	Title      string `yaml:"title"`
	Status     string `yaml:"status"`
	Priority   string `yaml:"priority"`
	Source     string `yaml:"source,omitempty"`
	SourceRef  string `yaml:"source_ref,omitempty"`
	ReportedBy string `yaml:"reported_by,omitempty"`
	Created    string `yaml:"created"`
}

// ReadMeta reads and parses the YAML frontmatter from a markdown file.
func ReadMeta(path string) (*SpecMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return ParseMeta(string(data))
}

// ParseMeta parses YAML frontmatter from markdown content.
func ParseMeta(content string) (*SpecMeta, error) {
	fm, err := extractFrontmatter(content)
	if err != nil {
		return nil, err
	}
	var meta SpecMeta
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}
	return &meta, nil
}

// ReadTriageMeta reads and parses triage frontmatter.
func ReadTriageMeta(path string) (*TriageMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return ParseTriageMeta(string(data))
}

// ParseTriageMeta parses triage YAML frontmatter.
func ParseTriageMeta(content string) (*TriageMeta, error) {
	fm, err := extractFrontmatter(content)
	if err != nil {
		return nil, err
	}
	var meta TriageMeta
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return nil, fmt.Errorf("parsing triage frontmatter: %w", err)
	}
	return &meta, nil
}

// WriteMeta updates the frontmatter in a file, preserving the body content.
func WriteMeta(path string, meta *SpecMeta) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	meta.Updated = time.Now().Format("2006-01-02")

	newContent, err := replaceFrontmatter(string(data), meta)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(newContent), 0o644)
}

// extractFrontmatter returns the YAML content between --- delimiters.
func extractFrontmatter(content string) (string, error) {
	if !strings.HasPrefix(content, "---") {
		return "", fmt.Errorf("no frontmatter found: file does not start with ---")
	}

	// Find the closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", fmt.Errorf("no closing --- found for frontmatter")
	}

	return strings.TrimSpace(rest[:idx]), nil
}

// replaceFrontmatter replaces the YAML frontmatter in content.
func replaceFrontmatter(content string, meta interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("marshalling frontmatter: %w", err)
	}

	// Find the end of existing frontmatter
	if !strings.HasPrefix(content, "---") {
		return "", fmt.Errorf("no frontmatter to replace")
	}
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", fmt.Errorf("no closing --- found")
	}

	body := rest[idx+4:] // skip \n---
	return "---\n" + string(yamlBytes) + "---" + body, nil
}

// Body returns the content after the frontmatter.
func Body(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return content
	}
	body := rest[idx+4:]
	return strings.TrimLeft(body, "\n")
}
