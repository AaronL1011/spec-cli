package dashboard

import (
	"testing"
	"time"

	"github.com/nexl/spec-cli/internal/store"
)

func TestPendingCount_NilDB(t *testing.T) {
	count := PendingCount(nil)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestPendingCount_EmptyCache(t *testing.T) {
	db, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	count := PendingCount(db)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestPendingCount_WithCachedData(t *testing.T) {
	db, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	data := `{"do":[{"spec_id":"SPEC-001"},{"spec_id":"SPEC-002"}],"review":[{"spec_id":"PR #10"}]}`
	if err := db.CacheSet("dashboard:data", data, 300); err != nil {
		t.Fatal(err)
	}

	count := PendingCount(db)
	if count != 3 {
		t.Errorf("expected 3 (2 do + 1 review), got %d", count)
	}
}

func TestTimeAgo(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{10 * time.Second, "just now"},
		{30 * time.Minute, "30m ago"},
		{3 * time.Hour, "3h ago"},
		{48 * time.Hour, "2d ago"},
	}
	for _, tt := range tests {
		got := timeAgo(time.Now().Add(-tt.d))
		if got != tt.want {
			t.Errorf("timeAgo(-%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestTruncStr(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"this is very long text", 10, "this is..."},
		{"exact", 5, "exact"},
	}
	for _, tt := range tests {
		got := truncStr(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("truncStr(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
		}
	}
}
