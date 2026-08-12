package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestKey(t *testing.T) {
	// Deterministic md5-based key with a stable prefix.
	if got := Key("stdict", "hello"); got != "stdict_5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("Key = %q", got)
	}
	// A word containing '/' still yields a filesystem-safe key.
	if got := Key("stdict", "a/b"); len(got) != len("stdict_")+32 {
		t.Fatalf("unexpected key length for slash word: %q", got)
	}
}

func TestReadWriteFreshness(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())

	if err := Write("k", []byte(`{"v":1}`)); err != nil {
		t.Fatal(err)
	}

	// Within maxAge -> hit.
	if data, ok := Read("k", time.Hour); !ok || string(data) != `{"v":1}` {
		t.Fatalf("fresh read failed: %q ok=%v", data, ok)
	}

	// maxAge 0 -> never expires, always a hit when present.
	if _, ok := Read("k", 0); !ok {
		t.Fatal("maxAge=0 should always hit when file exists")
	}

	// Stale (maxAge shorter than age) -> miss.
	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(path("k"), old, old)
	if _, ok := Read("k", time.Hour); ok {
		t.Fatal("stale entry should miss")
	}

	// Missing key -> miss.
	if _, ok := Read("nope", time.Hour); ok {
		t.Fatal("missing key should miss")
	}
}

func TestPrune(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())

	// Two per-query entries: one fresh, one stale.
	fresh := Key("stdict", "fresh")
	stale := Key("stdict", "stale")
	if err := Write(fresh, []byte("f")); err != nil {
		t.Fatal(err)
	}
	if err := Write(stale, []byte("s")); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path(stale), old, old); err != nil {
		t.Fatal(err)
	}

	// A fixed-key entry under a different prefix, even though it is old, must
	// survive: Prune only touches files carrying the requested prefix.
	if err := Write("__update_info", []byte("keep")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path("__update_info"), old, old); err != nil {
		t.Fatal(err)
	}

	if err := Prune("stdict", time.Hour); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path(stale)); !os.IsNotExist(err) {
		t.Fatalf("stale stdict entry should have been removed, stat err=%v", err)
	}
	if _, ok := Read(fresh, time.Hour); !ok {
		t.Fatal("fresh stdict entry should survive prune")
	}
	if _, ok := Read("__update_info", 0); !ok {
		t.Fatal("entry under a different prefix must survive prune")
	}
}

func TestPruneMissingDir(t *testing.T) {
	// A cache directory that was never created is not an error.
	t.Setenv("alfred_workflow_cache", filepath.Join(t.TempDir(), "does-not-exist"))
	if err := Prune("stdict", time.Hour); err != nil {
		t.Fatalf("Prune on missing dir = %v", err)
	}
}

func TestCachedUsesLoaderOnce(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())

	calls := 0
	loader := func() ([]byte, error) {
		calls++
		return []byte("payload"), nil
	}

	for i := 0; i < 3; i++ {
		data, err := Cached("k", time.Hour, loader)
		if err != nil || string(data) != "payload" {
			t.Fatalf("Cached returned %q err=%v", data, err)
		}
	}
	if calls != 1 {
		t.Fatalf("loader called %d times, want 1", calls)
	}

	// Nil loader on a miss returns no data without error.
	if data, err := Cached("missing", time.Hour, nil); err != nil || data != nil {
		t.Fatalf("nil loader miss = %q err=%v", data, err)
	}
}
