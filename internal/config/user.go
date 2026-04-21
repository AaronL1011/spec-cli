package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// UserConfig represents the ~/.spec/config.yaml personal config file.
type UserConfig struct {
	User struct {
		OwnerRole string `yaml:"owner_role"`
		Name      string `yaml:"name"`
		Handle    string `yaml:"handle"`
	} `yaml:"user"`

	Preferences PreferencesConfig `yaml:"preferences"`
}

// PreferencesConfig holds personal preferences.
type PreferencesConfig struct {
	Editor            string   `yaml:"editor"`
	DashboardSections []string `yaml:"dashboard_sections"`
	StandupAutoPost   bool     `yaml:"standup_auto_post"`
	AIDrafts          *bool    `yaml:"ai_drafts,omitempty"`
}

// AIDraftsEnabled returns whether AI drafts are enabled.
// Defaults to true if not explicitly set.
func (p PreferencesConfig) AIDraftsEnabled() bool {
	if p.AIDrafts == nil {
		return true
	}
	return *p.AIDrafts
}

// UserConfigDir returns the path to the ~/.spec/ directory.
func UserConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".spec")
	}
	return filepath.Join(home, ".spec")
}

// UserConfigPath returns the path to ~/.spec/config.yaml.
func UserConfigPath() string {
	return filepath.Join(UserConfigDir(), "config.yaml")
}

// LoadUserConfig reads and parses the user config file.
func LoadUserConfig(path string) (*UserConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading user config %s: %w", path, err)
	}
	data = interpolateEnvVars(data)

	var cfg UserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing user config %s: %w", path, err)
	}

	// Defaults
	if cfg.Preferences.Editor == "" {
		if editor := os.Getenv("EDITOR"); editor != "" {
			cfg.Preferences.Editor = editor
		} else {
			cfg.Preferences.Editor = "vi"
		}
	}
	if len(cfg.Preferences.DashboardSections) == 0 {
		cfg.Preferences.DashboardSections = []string{"do", "review", "incoming", "blocked"}
	}
	return &cfg, nil
}

// LoadUserConfigWithDefaults loads user config or returns defaults if file doesn't exist.
func LoadUserConfigWithDefaults() (*UserConfig, string) {
	path := UserConfigPath()
	cfg, err := LoadUserConfig(path)
	if err != nil {
		// Return default config
		cfg = &UserConfig{}
		cfg.Preferences.Editor = os.Getenv("EDITOR")
		if cfg.Preferences.Editor == "" {
			cfg.Preferences.Editor = "vi"
		}
		cfg.Preferences.DashboardSections = []string{"do", "review", "incoming", "blocked"}
		return cfg, path
	}
	return cfg, path
}

// WriteUserConfig writes a user config to disk.
func WriteUserConfig(path string, cfg *UserConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling user config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing user config %s: %w", path, err)
	}
	return nil
}

// ValidRoles returns the valid owner roles.
func ValidRoles() []string {
	return []string{"pm", "tl", "designer", "qa", "engineer"}
}

// IsValidRole checks if a role string is valid.
func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if r == role {
			return true
		}
	}
	return false
}
