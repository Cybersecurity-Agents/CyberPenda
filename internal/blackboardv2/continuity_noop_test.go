package blackboardv2_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"pentest/internal/blackboardv2"
	"testing"
)

func TestUnchangedRuntimeBatchDoesNotRewriteWorkingSnapshot(t *testing.T) {
	f := newContinuityFixture(t)
	t.Cleanup(func() { _ = f.db.Close() })
	launch := f.launch(t)
	if err := f.continuity.MaterializeWorkingSnapshot(context.Background(), launch.Continuation.ID); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.runtimeRoot, f.task.ID, "workdir", ".pentest", "blackboard.json")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`CREATE TRIGGER reject_unchanged_working_snapshot BEFORE UPDATE ON blackboard_v2_continuation_state WHEN NEW.last_acknowledged_revision=OLD.last_acknowledged_revision BEGIN SELECT RAISE(ABORT, 'unchanged Working Snapshot rewrite'); END`); err != nil {
		t.Fatal(err)
	}
	result, err := f.board.ApplyForContinuation(context.Background(), f.project.ID, launch.Continuation.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "unchanged-working-snapshot", Changes: []blackboardv2.Change{}})
	if err != nil {
		t.Fatalf("unchanged Runtime batch: %v", err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("unchanged batch rewrote Working Snapshot file")
	}
	if result.Revision != launch.Revision {
		t.Fatalf("revision = %d, want %d", result.Revision, launch.Revision)
	}
}

func TestUnchangedRuntimeBatchRepairsFailedWorkingSnapshotPublication(t *testing.T) {
	f := newContinuityFixture(t)
	t.Cleanup(func() { _ = f.db.Close() })
	launch := f.launch(t)
	if err := f.continuity.MaterializeWorkingSnapshot(context.Background(), launch.Continuation.ID); err != nil {
		t.Fatal(err)
	}
	f.board.SetSnapshotPublicationInjector(func(point blackboardv2.SnapshotPublicationPoint, id string) error {
		if point == blackboardv2.SnapshotPublicationAfterCommit {
			return errors.New("publication failed")
		}
		return nil
	})
	_, err := f.board.ApplyForContinuation(context.Background(), f.project.ID, launch.Continuation.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "failed-publication", Changes: []blackboardv2.Change{{Op: "create", Key: "entity:committed", Type: "entity", Record: blackboardv2.EntityRecord{Status: "active", Kind: "host", Name: "Committed", ScopeStatus: "in_scope"}}}})
	if err == nil {
		t.Fatal("publication should fail after commit")
	}
	f.board.SetSnapshotPublicationInjector(nil)
	_, err = f.board.ApplyForContinuation(context.Background(), f.project.ID, launch.Continuation.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "repair-with-another-key", Changes: []blackboardv2.Change{}})
	if err != nil {
		t.Fatal(err)
	}
	working, err := f.continuity.ReadWorkingSnapshot(context.Background(), launch.Continuation.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(f.runtimeRoot, f.task.ID, "workdir", ".pentest", "blackboard.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, working.Bytes) {
		t.Fatal("unchanged batch left stale Working Snapshot file")
	}
}
