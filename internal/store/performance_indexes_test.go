package store

import (
	"path/filepath"
	"testing"
)

func TestOpenAddsPerformanceIndexesToExistingBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var checksum string
	if err := db.QueryRow(`SELECT checksum FROM schema_migrations`).Scan(&checksum); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"idx_session_events_permissions", "idx_session_events_session_kind_seq"} {
		if _, err := db.Exec(`DROP INDEX ` + name); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("open baseline without optional indexes: %v", err)
	}
	defer reopened.Close()
	var got string
	if err := reopened.QueryRow(`SELECT checksum FROM schema_migrations`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != checksum {
		t.Fatalf("baseline checksum changed: %s -> %s", checksum, got)
	}
	for _, name := range []string{"idx_session_events_permissions", "idx_session_events_session_kind_seq"} {
		var count int
		if err := reopened.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("performance index %s is missing", name)
		}
	}
}
