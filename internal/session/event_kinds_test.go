package session

import (
	"path/filepath"
	"pentest/internal/store"
	"strings"
	"testing"
)

func TestConversationDoesNotDecodeUnrelatedRuntimePayloads(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "session.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(db, filepath.Join(t.TempDir(), "sessions"))
	created, err := service.Create(CreateRequest{Input: "Review the target"})
	if err != nil {
		t.Fatal(err)
	}
	// An unrelated retained payload must not enter the Conversation decoder.
	_, err = db.Exec(`INSERT INTO session_events(id,session_id,seq,kind,payload_json,created_at) VALUES('unrelated-output',?,2,'runtime_output','not-json','2026-01-01T00:00:00Z')`, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	events, err := service.Conversation(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Kind != EventKindConversation {
		t.Fatalf("Conversation = %#v", events)
	}
}

func TestSessionKindProjectionUsesIndexAfterExistingStoreReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP INDEX idx_session_events_session_kind_seq`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`EXPLAIN QUERY PLAN SELECT id,session_id,seq,kind,payload_json,created_at FROM session_events WHERE session_id=? AND kind IN (?) ORDER BY seq ASC`, "session", EventKindConversation)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	indexed := false
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(detail, "idx_session_events_session_kind_seq") && strings.Contains(detail, "kind=?") {
			indexed = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !indexed {
		t.Fatal("Session kind projection does not use its kind index")
	}
}
