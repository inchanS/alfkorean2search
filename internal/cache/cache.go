// Package cache is a small on-disk cache replacing wf.cached_data / cache_data.
// Entries are JSON files under Alfred's per-workflow cache directory, with
// freshness judged by file mtime.
package cache

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"
)

// dir returns Alfred's workflow cache directory, or a temp fallback when the
// workflow env var is absent (e.g. local runs and tests).
func dir() string {
	if d := os.Getenv("alfred_workflow_cache"); d != "" {
		return d
	}
	return filepath.Join(os.TempDir(), "alfkorean2search-cache")
}

func path(key string) string {
	return filepath.Join(dir(), key+".json")
}

// Key builds a filesystem-safe cache key: prefix + md5(word). The word may
// contain '/' or other path separators, so it is always hashed.
func Key(prefix, word string) string {
	sum := md5.Sum([]byte(word))
	return prefix + "_" + hex.EncodeToString(sum[:])
}

// Read returns the cached bytes for key when present and still fresh. A maxAge
// of 0 means "never expires": any existing entry is returned.
func Read(key string, maxAge time.Duration) ([]byte, bool) {
	p := path(key)
	fi, err := os.Stat(p)
	if err != nil {
		return nil, false
	}
	if maxAge > 0 && time.Since(fi.ModTime()) > maxAge {
		return nil, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Write stores data under key, creating the cache directory as needed.
func Write(key string, data []byte) error {
	if err := os.MkdirAll(dir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path(key), data, 0o644)
}

// Cached returns a fresh cached entry for key, or invokes loader, stores its
// result and returns it. When loader is nil and the entry is missing/stale,
// it returns (nil, nil) — mirroring cached_data(key, None, ...).
func Cached(key string, maxAge time.Duration, loader func() ([]byte, error)) ([]byte, error) {
	if data, ok := Read(key, maxAge); ok {
		return data, nil
	}
	if loader == nil {
		return nil, nil
	}
	data, err := loader()
	if err != nil {
		return nil, err
	}
	if err := Write(key, data); err != nil {
		return nil, err
	}
	return data, nil
}
