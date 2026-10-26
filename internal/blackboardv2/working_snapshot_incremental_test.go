package blackboardv2_test

import (
	"bytes"
	"context"
	"pentest/internal/blackboardv2"
	"testing"
)

func TestWorkingSnapshotChangesMatchFullProjectionAcrossOwnAndPeerBatches(t *testing.T) {
	f := newContinuityFixture(t)
	defer f.db.Close()
	ctx := context.Background()
	launch := f.launch(t)
	check := func() {
		t.Helper()
		full, err := f.board.ProjectRuntimeSnapshot(ctx, f.project.ID)
		if err != nil {
			t.Fatal(err)
		}
		working, err := f.continuity.ReadWorkingSnapshot(ctx, launch.Continuation.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(working.Bytes, full.Bytes) || working.LastAcknowledgedRevision != full.Snapshot.Revision {
			t.Fatalf("Working Snapshot differs from full projection\nworking=%s\nfull=%s", working.Bytes, full.Bytes)
		}
	}
	apply := func(id string, changes ...blackboardv2.Change) {
		t.Helper()
		if _, err := f.board.ApplyForContinuation(ctx, f.project.ID, launch.Continuation.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: id, Changes: changes}); err != nil {
			t.Fatal(err)
		}
		check()
	}
	entity := func(key, name string) blackboardv2.Change {
		return blackboardv2.Change{Op: "create", Key: key, Type: "entity", Record: blackboardv2.EntityRecord{Status: "active", Kind: "host", Name: name, ScopeStatus: "in_scope"}}
	}
	apply("seed", entity("entity:a", "a"), entity("entity:b", "b"))
	name := "new a"
	apply("update", blackboardv2.Change{Op: "update", Key: "entity:a", Type: "entity", Version: 1, Record: blackboardv2.EntityPatch{Name: &name}})
	apply("relate", blackboardv2.Change{Op: "relate", From: "entity:a", Relation: "part_of", To: "entity:b"})
	apply("retire", blackboardv2.Change{Op: "transition", Key: "entity:b", Version: 1, Status: "retired", ResolutionSummary: "Retired after inspection"})
	apply("replacement", entity("entity:new", "new"), blackboardv2.Change{Op: "supersede", Replacement: "entity:new", Replaced: "entity:a", ReplacedVersion: 2})
	// Another writer advances the Project while this Continuation retains its old
	// Working Snapshot. The next own batch must include that peer's full state.
	if _, err := f.board.Apply(ctx, f.project.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "peer", Changes: []blackboardv2.Change{entity("entity:peer", "peer")}}); err != nil {
		t.Fatal(err)
	}
	apply("after-peer", entity("entity:own", "own"))
	apply("no-changes", []blackboardv2.Change{}...)
}
