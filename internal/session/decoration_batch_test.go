package session

import (
	"path/filepath"
	"pentest/internal/store"
	"testing"
)

func TestDecorationDataForOwnersKeepsLatestAndOwnerBoundaries(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewService(db, filepath.Join(t.TempDir(), "sessions"))
	a, err := s.Create(CreateRequest{Input: "one"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(CreateRequest{Input: "two"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.CreateContinuation(a.ID, "profile", "codex", RunnerSandbox, map[string]any{"snapshot_version": 1, "runtime_plugin_id": "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateContinuationStatus(first.ID, RuntimeStatusCompleted); err != nil {
		t.Fatal(err)
	}
	latest, err := s.CreateContinuation(a.ID, "profile", "pi", RunnerSandbox, map[string]any{"snapshot_version": 1, "runtime_plugin_id": "pi"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{a.ID, b.ID} {
		if _, err = s.AppendEvent(id, EventKindPermission, EventPayload{"phase": "provider_permission_requested", "permission_request_id": "same-id"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.AppendEvent(a.ID, EventKindRuntimeOutput, EventPayload{"text": "unrelated output"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.DecorationDataForOwners([]string{a.ID, a.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("owner count = %d", len(got))
	}
	data := got[a.ID]
	if data.Latest == nil || data.Active == nil || data.Latest.ID != latest.ID || data.Active.ID != latest.ID {
		t.Fatalf("continuations = %#v", data)
	}
	if data.Config["runtime_plugin_id"] != "pi" {
		t.Fatalf("config = %#v", data.Config)
	}
	if len(data.PermissionEvents) != 1 || data.PermissionEvents[0].SessionID != a.ID {
		t.Fatalf("permissions = %#v", data.PermissionEvents)
	}
	empty, err := s.DecorationDataForOwners(nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty = %#v,%v", empty, err)
	}
}
