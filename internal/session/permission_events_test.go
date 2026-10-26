package session

import (
	"path/filepath"
	"pentest/internal/store"
	"strings"
	"testing"
)

func TestPermissionEventsSelectsOnlyPermissionChangesInOrder(t *testing.T) {
	root := t.TempDir()
	db, err := store.Open(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewService(db, filepath.Join(root, "sessions"))
	created, err := s.Create(CreateRequest{Input: "Inspect the target"})
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range []EventPayload{
		{"phase": "provider_permission_requested", "permission_request_id": "p1"},
		{"phase": "runtime_started"},
		{"phase": "provider_permission_requested", "permission_request_id": "p2"},
		{"phase": "provider_permission_response_applied", "permission_request_id": "p1"},
		{"phase": "provider_permission_response_failed", "permission_request_id": "p2"},
	} {
		if _, err := s.AppendEvent(created.ID, EventKindLifecycle, payload); err != nil {
			t.Fatal(err)
		}
	}
	events, err := s.PermissionEvents(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("permission events = %d, want 3", len(events))
	}
	for i, want := range []string{"p1", "p2", "p1"} {
		if events[i].Payload["permission_request_id"] != want {
			t.Fatalf("event %d = %#v", i, events[i])
		}
		if i > 0 && events[i].Seq <= events[i-1].Seq {
			t.Fatal("events are not ordered")
		}
	}
	if _, err := s.PermissionEvents("missing"); err != ErrNotFound {
		t.Fatalf("missing Session error = %v", err)
	}
}

func TestPermissionEventsUsesPermissionIndex(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`EXPLAIN QUERY PLAN SELECT id,session_id,seq,kind,payload_json,created_at
 FROM session_events WHERE session_id=? AND kind IN ('permission','lifecycle')
 AND json_extract(payload_json,'$.permission_request_id') IS NOT NULL
 AND json_extract(payload_json,'$.phase') IN ('provider_permission_requested','provider_permission_response_applied')
 ORDER BY seq ASC`, "session-test")
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
		indexed = indexed || strings.Contains(detail, "idx_session_events_permissions")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !indexed {
		t.Fatal("permission history query does not use the partial index")
	}
}
