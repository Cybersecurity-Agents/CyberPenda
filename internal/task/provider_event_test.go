package task_test

import (
	"errors"
	"pentest/internal/project"
	"pentest/internal/task"
	"testing"
)

func TestProviderEventUsesCurrentContinuationAndRejectsMissingTask(t *testing.T) {
	db := newStore(t)
	projects := project.NewService(db)
	service := task.NewService(db, projects)
	p, err := projects.Create("Provider events", "", project.Scope{}, project.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := service.Create(task.CreateRequest{ProjectID: p.ID, Type: task.TypePentest, Goal: "g", Runner: task.RunnerSandbox})
	if err != nil {
		t.Fatal(err)
	}
	appendEvent := func(want string) {
		t.Helper()
		event, err := service.AppendProviderEvent(owner.ID, task.EventKindRuntimeOutput, task.EventPayload{"text": "unchanged"})
		if err != nil || event.ContinuationID != want || event.Payload["text"] != "unchanged" {
			t.Fatalf("event = %#v, %v; Continuation = %q", event, err, want)
		}
	}
	appendEvent("")
	first, err := service.CreateContinuation(owner.ID, "", "fake", task.RunnerSandbox)
	if err != nil {
		t.Fatal(err)
	}
	appendEvent(first.ID)
	if _, err := service.UpdateContinuationStatus(first.ID, task.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	appendEvent("")
	second, err := service.CreateContinuation(owner.ID, "", "fake", task.RunnerSandbox)
	if err != nil {
		t.Fatal(err)
	}
	appendEvent(second.ID)
	if _, err := service.AppendProviderEvent("missing", task.EventKindRuntimeOutput, nil); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("missing owner error = %v", err)
	}
	if _, err := db.Exec(`UPDATE tasks SET deleted_at='2026-10-26T00:00:00Z' WHERE id=?`, owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendProviderEvent(owner.ID, task.EventKindRuntimeOutput, nil); !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("deleted owner error = %v", err)
	}
	events, err := service.Events(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, event := range events {
		if event.Seq != i+1 {
			t.Fatalf("unordered event: %#v", event)
		}
	}
}
