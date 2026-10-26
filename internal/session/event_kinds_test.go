package session

import (
	"path/filepath"
	"pentest/internal/store"
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
