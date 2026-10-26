package task_test

import (
	"pentest/internal/project"
	"pentest/internal/task"
	"testing"
)

func TestRuntimeControlEventsExcludeOutputAndOldRecovery(t *testing.T) {
	db := newStore(t)
	projects := project.NewService(db)
	svc := task.NewService(db, projects)
	proj, err := projects.Create("P", "", project.Scope{}, project.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	created, err := svc.Create(task.CreateRequest{ProjectID: proj.ID, Type: task.TypePentest, Goal: "g", RuntimeProfileID: "p", Runner: task.RunnerSandbox})
	if err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"provider_permission_requested", "provider_session_recovery_required", "started", "provider_permission_response_applied", "provider_permission_requested"} {
		if _, err := svc.AppendEvent(created.ID, task.EventKindLifecycle, task.EventPayload{"phase": phase}); err != nil {
			t.Fatal(err)
		}
	}
	output, err := svc.AppendEvent(created.ID, task.EventKindRuntimeOutput, task.EventPayload{"text": "large output"})
	if err != nil {
		t.Fatal(err)
	}
	// An unrelated payload must not be decoded by the controls read.
	if _, err := db.Exec(`UPDATE task_events SET payload_json='not JSON' WHERE id=?`, output.ID); err != nil {
		t.Fatal(err)
	}
	events, err := svc.RuntimeControlEvents(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("got %d events, want three permission events and latest recovery boundary", len(events))
	}
	for _, event := range events {
		if event.Payload["phase"] == "provider_session_recovery_required" {
			t.Fatal("old recovery included")
		}
	}
}

func TestLatestRuntimeConfigSkipsOlderPayloads(t *testing.T) {
	db := newStore(t)
	projects := project.NewService(db)
	svc := task.NewService(db, projects)
	proj, err := projects.Create("P", "", project.Scope{}, project.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	created, err := svc.Create(task.CreateRequest{ProjectID: proj.ID, Type: task.TypePentest, Goal: "g", RuntimeProfileID: "p", Runner: task.RunnerSandbox})
	if err != nil {
		t.Fatal(err)
	}
	if found, err := svc.LatestRuntimeConfig(created.ID); err != nil || found != nil {
		t.Fatalf("empty latest: %v %v", found, err)
	}
	old, err := svc.RecordRuntimeConfig(created.ID, "p", map[string]any{"model": "old"})
	if err != nil {
		t.Fatal(err)
	}
	latest, err := svc.RecordRuntimeConfig(created.ID, "p", map[string]any{"model": "new"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE task_runtime_config_versions SET config_json='not JSON' WHERE id=?`, old.ID); err != nil {
		t.Fatal(err)
	}
	found, err := svc.LatestRuntimeConfig(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.ID != latest.ID || found.Config["model"] != "new" {
		t.Fatalf("wrong latest: %#v", found)
	}
}
