package store

import (
	"database/sql"
	"fmt"
	"time"
)

// SyncStateEntry records the last hash synced for one spec section and direction.
type SyncStateEntry struct {
	SpecID    string
	Section   string
	Direction string
	Hash      string
	SyncedAt  time.Time
}

// SyncStateGet returns the last synced hash for a spec section and direction.
func (db *DB) SyncStateGet(specID, section, direction string) (*SyncStateEntry, error) {
	var entry SyncStateEntry
	var syncedAt int64
	err := db.conn.QueryRow(
		`SELECT spec_id, section, direction, hash, synced_at
		 FROM sync_state
		 WHERE spec_id = ? AND section = ? AND direction = ?`,
		specID, section, direction,
	).Scan(&entry.SpecID, &entry.Section, &entry.Direction, &entry.Hash, &syncedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sync state get %s/%s/%s: %w", specID, section, direction, err)
	}
	entry.SyncedAt = time.Unix(syncedAt, 0)
	return &entry, nil
}

// SyncStateSet upserts the last synced hash for a spec section and direction.
func (db *DB) SyncStateSet(specID, section, direction, hash string) error {
	_, err := db.conn.Exec(
		`INSERT INTO sync_state (spec_id, section, direction, hash, synced_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(spec_id, section, direction)
		 DO UPDATE SET hash = excluded.hash, synced_at = excluded.synced_at`,
		specID, section, direction, hash, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("sync state set %s/%s/%s: %w", specID, section, direction, err)
	}
	return nil
}

// SyncStateForSpec returns all sync state entries for a spec keyed by section and direction.
func (db *DB) SyncStateForSpec(specID string) (map[string]map[string]SyncStateEntry, error) {
	rows, err := db.conn.Query(
		`SELECT spec_id, section, direction, hash, synced_at
		 FROM sync_state
		 WHERE spec_id = ?`,
		specID,
	)
	if err != nil {
		return nil, fmt.Errorf("sync state for spec %s: %w", specID, err)
	}
	defer func() { _ = rows.Close() }()

	entries := make(map[string]map[string]SyncStateEntry)
	for rows.Next() {
		var entry SyncStateEntry
		var syncedAt int64
		if err := rows.Scan(&entry.SpecID, &entry.Section, &entry.Direction, &entry.Hash, &syncedAt); err != nil {
			return nil, fmt.Errorf("scanning sync state: %w", err)
		}
		entry.SyncedAt = time.Unix(syncedAt, 0)
		if entries[entry.Section] == nil {
			entries[entry.Section] = make(map[string]SyncStateEntry)
		}
		entries[entry.Section][entry.Direction] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sync state: %w", err)
	}
	return entries, nil
}
