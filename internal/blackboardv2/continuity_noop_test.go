package blackboardv2_test

import (
	"context"
	"pentest/internal/blackboardv2"
	"testing"
)

func TestUnchangedRuntimeBatchDoesNotRewriteWorkingSnapshot(t *testing.T) {
	f := newContinuityFixture(t)
	t.Cleanup(func() { _ = f.db.Close() })
	launch := f.launch(t)
	if _, err := f.db.Exec(`CREATE TRIGGER reject_unchanged_working_snapshot BEFORE UPDATE ON blackboard_v2_continuation_state WHEN NEW.last_acknowledged_revision=OLD.last_acknowledged_revision BEGIN SELECT RAISE(ABORT, 'unchanged Working Snapshot rewrite'); END`); err != nil {
		t.Fatal(err)
	}
	result, err := f.board.ApplyForContinuation(context.Background(), f.project.ID, launch.Continuation.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "unchanged-working-snapshot", Changes: []blackboardv2.Change{}})
	if err != nil {
		t.Fatalf("unchanged Runtime batch: %v", err)
	}
	if result.Revision != launch.Revision {
		t.Fatalf("revision = %d, want %d", result.Revision, launch.Revision)
	}
}
