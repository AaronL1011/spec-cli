package store

import (
	"database/sql"
	"fmt"
	"time"
)

// CacheGet retrieves a cached value by key. Returns the value and whether
// the cache entry is still fresh (within TTL). Returns "", false if not found.
func (db *DB) CacheGet(key string) (value string, fresh bool, err error) {
	var fetchedAt, ttl int64
	err = db.conn.QueryRow(
		"SELECT value, fetched_at, ttl FROM cache WHERE key = ?", key,
	).Scan(&value, &fetchedAt, &ttl)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("cache get %q: %w", key, err)
	}

	fresh = time.Now().Unix()-fetchedAt < ttl
	return value, fresh, nil
}

// CacheSet stores a value in the cache with the given TTL in seconds.
func (db *DB) CacheSet(key, value string, ttlSeconds int) error {
	_, err := db.conn.Exec(
		`INSERT INTO cache (key, value, fetched_at, ttl)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, fetched_at=excluded.fetched_at, ttl=excluded.ttl`,
		key, value, time.Now().Unix(), ttlSeconds,
	)
	if err != nil {
		return fmt.Errorf("cache set %q: %w", key, err)
	}
	return nil
}

// CacheDelete removes a cache entry.
func (db *DB) CacheDelete(key string) error {
	_, err := db.conn.Exec("DELETE FROM cache WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("cache delete %q: %w", key, err)
	}
	return nil
}

// CacheClear removes all cache entries.
func (db *DB) CacheClear() error {
	_, err := db.conn.Exec("DELETE FROM cache")
	return err
}
