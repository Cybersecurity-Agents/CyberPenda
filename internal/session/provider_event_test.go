package session

import (
	"errors"
	"path/filepath"
	"pentest/internal/store"
	"testing"
)

func TestProviderEventUsesServerBindingWithoutMutatingPayload(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(db, filepath.Join(t.TempDir(), "sessions"))
	owner, err := service.Create(CreateRequest{Input: "events"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.Create(CreateRequest{Input: "other owner"})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := service.CreateContinuation(other.ID, "", "fake", RunnerSandbox)
	if err != nil {
		t.Fatal(err)
	}
	input := EventPayload{"text": "unchanged", "continuation_id": foreign.ID}
	appendEvent := func(want string) {
		t.Helper()
		event, err := service.AppendProviderEvent(owner.ID, EventKindRuntimeOutput, input)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := event.Payload["continuation_id"].(string)
		if got != want || event.Payload["text"] != "unchanged" || input["continuation_id"] != foreign.ID {
			t.Fatalf("event = %#v; input = %#v; want Continuation %q", event, input, want)
		}
	}
	appendEvent("")
	first, err := service.CreateContinuation(owner.ID, "", "fake", RunnerSandbox)
	if err != nil {
		t.Fatal(err)
	}
	appendEvent(first.ID)
	if _, err := service.UpdateContinuationStatus(first.ID, RuntimeStatusCompleted); err != nil {
		t.Fatal(err)
	}
	appendEvent("")
	second, err := service.CreateContinuation(owner.ID, "", "fake", RunnerSandbox)
	if err != nil {
		t.Fatal(err)
	}
	appendEvent(second.ID)
	if _, err := service.AppendProviderEvent("missing", EventKindRuntimeOutput, input); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing owner: %v", err)
	}
	if _, err := db.Exec(`UPDATE sessions SET lifecycle='archived' WHERE id=?`, owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendProviderEvent(owner.ID, EventKindRuntimeOutput, input); !errors.Is(err, ErrSessionNotOpen) {
		t.Fatalf("closed owner: %v", err)
	}
}
