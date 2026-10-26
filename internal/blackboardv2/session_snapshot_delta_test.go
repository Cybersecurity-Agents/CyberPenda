package blackboardv2

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"pentest/internal/session"
	"pentest/internal/store"
)

func TestSessionSnapshotDeltaMatchesFullProjection(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sessions := session.NewService(db, filepath.Join(t.TempDir(), "sessions"))
	found, err := sessions.Create(session.CreateRequest{Input: "Snapshot delta"})
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(db)
	_, err = s.ApplyForSession(ctx, found.ID, ChangeBatch{Schema: changeBatchSchema, IdempotencyKey: "seed", Changes: []Change{
		{Op: "create", Key: "entity:old", Type: "entity", Record: EntityRecord{Status: "active", Kind: "host", Name: "old"}},
		{Op: "create", Key: "entity:keep", Type: "entity", Record: EntityRecord{Status: "active", Kind: "host", Name: "keep"}},
		{Op: "create", Key: "fact:old", Type: "fact", Record: FactRecord{Category: "note", Summary: "old", Confidence: "tentative"}},
		{Op: "relate", From: "fact:old", Relation: "about", To: "entity:old"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE blackboard_v2_session_records SET version=2,record_json=? WHERE session_id=? AND key='fact:old'`, `{"category":"note","summary":"new","confidence":"confirmed"}`, found.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`DELETE FROM blackboard_v2_session_records WHERE session_id=? AND key='entity:old'`, found.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`DELETE FROM blackboard_v2_session_relationships WHERE session_id=?`, found.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE blackboard_v2_session_state SET revision=6 WHERE session_id=?`, found.ID); err != nil {
		t.Fatal(err)
	}
	changes := semanticChangeSetResult{Revision: 6, ChangedRecords: map[string]int{"entity:old": 2, "fact:old": 2}}
	got, err := s.sessionSnapshotWithChangesTx(ctx, tx, found.ID, 4, changes)
	if err != nil {
		t.Fatal(err)
	}
	want, err := s.sessionRuntimeSnapshotTx(ctx, tx, found.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		a, _ := json.Marshal(got)
		b, _ := json.Marshal(want)
		t.Fatalf("delta = %s\nfull = %s", a, b)
	}
	// A changed current relationship is loaded by identity; historical-only
	// supersedes tuples must not leak into the current Snapshot.
	if _, err = tx.Exec(`INSERT INTO blackboard_v2_session_relationships(session_id,from_key,relation,to_key,version,reason,created_at,updated_at) VALUES(?,'fact:old','about','entity:keep',1,'context','2026-01-01','2026-01-01')`, found.ID); err != nil {
		t.Fatal(err)
	}
	changes.ChangedRelations = map[string]RelationVersionTuple{
		"current": {"fact:old", "about", "entity:keep", 1},
		"history": {"entity:keep", "supersedes", "entity:old", 1},
	}
	got, err = s.sessionSnapshotWithChangesTx(ctx, tx, found.ID, 4, changes)
	if err != nil {
		t.Fatal(err)
	}
	want, err = s.sessionRuntimeSnapshotTx(ctx, tx, found.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("relationship delta = %#v; full = %#v; err = %v", got, want, err)
	}
	// A stale cache must use a complete projection, rather than applying a
	// delta to a Snapshot from a different revision.
	fallback, err := s.sessionSnapshotWithChangesTx(ctx, tx, found.ID, 99, changes)
	if err != nil || !reflect.DeepEqual(fallback, want) {
		t.Fatalf("stale cache fallback = %#v, %v", fallback, err)
	}
}

func BenchmarkSessionSnapshotOneChangedFact(b *testing.B) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(b.TempDir(), "snapshot.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	sessions := session.NewService(db, filepath.Join(b.TempDir(), "sessions"))
	found, err := sessions.Create(session.CreateRequest{Input: "Snapshot delta benchmark"})
	if err != nil {
		b.Fatal(err)
	}
	s := NewService(db)
	const count = 250
	changes := make([]Change, 0, count)
	for i := 0; i < count; i++ {
		changes = append(changes, Change{Op: "create", Key: fmt.Sprintf("fact:%03d", i), Type: "fact", Record: FactRecord{Category: "note", Summary: "Compact summary", Confidence: "tentative", Body: strings.Repeat("body ", 400)}})
	}
	if _, err = s.ApplyForSession(ctx, found.ID, ChangeBatch{Schema: changeBatchSchema, IdempotencyKey: "seed", Changes: changes}); err != nil {
		b.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Rollback()
	delta := semanticChangeSetResult{Revision: count, ChangedRecords: map[string]int{"fact:000": 1}}
	for _, mode := range []string{"full", "delta"} {
		b.Run(mode, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var snapshot RuntimeSnapshot
				var err error
				if mode == "full" {
					snapshot, err = s.sessionRuntimeSnapshotTx(ctx, tx, found.ID)
				} else {
					snapshot, err = s.sessionSnapshotWithChangesTx(ctx, tx, found.ID, count, delta)
				}
				if err != nil {
					b.Fatal(err)
				}
				if _, err = json.Marshal(snapshot); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
