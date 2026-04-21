package slack

import "testing"

func TestExtractSpecID(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"Check out SPEC-042 for details", "SPEC-042"},
		{"SPEC-1 is ready", "SPEC-1"},
		{"No spec here", ""},
		{"SPEC- no digits", ""},
		{"Multiple SPEC-10 and SPEC-20", "SPEC-10"},
		{"Prefix text SPEC-999 suffix", "SPEC-999"},
	}
	for _, tt := range tests {
		got := extractSpecID(tt.text)
		if got != tt.want {
			t.Errorf("extractSpecID(%q) = %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestNormaliseChannel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"#platform", "platform"},
		{"platform", "platform"},
		{"#platform-standup", "platform-standup"},
	}
	for _, tt := range tests {
		got := normaliseChannel(tt.input)
		if got != tt.want {
			t.Errorf("normaliseChannel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"exact", 5, "exact"},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
