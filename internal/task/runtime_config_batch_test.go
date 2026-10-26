package task_test

import (
	"pentest/internal/project"
	"pentest/internal/task"
	"testing"
)

func TestLatestRuntimeConfigsUsesLatestVersionAndRequestedOwners(t *testing.T) {
	db := newStore(t)
	projects := project.NewService(db)
	s := task.NewService(db, projects)
	proj, err := projects.Create("P", "", project.Scope{}, project.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	one, err := s.Create(task.CreateRequest{ProjectID: proj.ID, Type: task.TypePentest, Goal: "one", Runner: task.RunnerSandbox})
	if err != nil {
		t.Fatal(err)
	}
	two, err := s.Create(task.CreateRequest{ProjectID: proj.ID, Type: task.TypePentest, Goal: "two", Runner: task.RunnerSandbox})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		id, owner, config string
		version           int
	}{{"one-old", one.ID, `{"runtime_plugin_id":"claude"}`, 1}, {"one-new", one.ID, `{"runtime_plugin_id":"codex"}`, 2}, {"two-config", two.ID, `{"runtime_plugin_id":"pi"}`, 1}} {
		_, err := db.Exec(`INSERT INTO task_runtime_config_versions(id,task_id,version,runtime_profile_id,config_json,created_at) VALUES(?,?,?,'',?,'2026-01-01T00:00:00Z')`, row.id, row.owner, row.version, row.config)
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.LatestRuntimeConfigs([]string{one.ID, "missing", one.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[one.ID]["runtime_plugin_id"] != "codex" {
		t.Fatalf("configs = %#v", got)
	}
	empty, err := s.LatestRuntimeConfigs(nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty = %#v, %v", empty, err)
	}
}
