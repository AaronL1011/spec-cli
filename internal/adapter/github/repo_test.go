package github

import "testing"

func TestSpecBranchPrefix(t *testing.T) {
	tests := []struct {
		specID string
		want   string
	}{
		{"SPEC-042", "spec-042/"},
		{"SPEC-1", "spec-1/"},
		{"spec-100", "spec-spec-100/"}, // already lowercase, double prefix — unlikely but deterministic
	}
	for _, tt := range tests {
		got := specBranchPrefix(tt.specID)
		if got != tt.want {
			t.Errorf("specBranchPrefix(%q) = %q, want %q", tt.specID, got, tt.want)
		}
	}
}

func TestExtractRepoFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://api.github.com/repos/my-org/auth-service", "auth-service"},
		{"https://api.github.com/repos/my-org/api-gateway", "api-gateway"},
		{"short", "short"},
	}
	for _, tt := range tests {
		got := extractRepoFromURL(tt.url)
		if got != tt.want {
			t.Errorf("extractRepoFromURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
