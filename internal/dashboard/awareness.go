package dashboard

import (
	"encoding/json"
	"fmt"

	"github.com/nexl/spec-cli/internal/store"
)

// PendingCount returns the number of items awaiting user attention.
// Reads from cache only — never blocks on network.
func PendingCount(db *store.DB) int {
	if db == nil {
		return 0
	}

	cached, _, err := db.CacheGet("dashboard:data")
	if err != nil || cached == "" {
		return 0
	}

	var data DashboardData
	if err := json.Unmarshal([]byte(cached), &data); err != nil {
		return 0
	}

	return len(data.Do) + len(data.Review)
}

// PrintAwarenessLine prints the passive "you have mail" indicator.
// Returns true if something was printed.
func PrintAwarenessLine(db *store.DB) bool {
	count := PendingCount(db)
	if count == 0 {
		return false
	}

	fmt.Printf("⚠ %d pending · run 'spec' for details\n", count)
	return true
}
