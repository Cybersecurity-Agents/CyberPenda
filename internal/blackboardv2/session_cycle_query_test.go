package blackboardv2

import (
	"context"
	"database/sql"
	"testing"
)

func TestSessionCycleChecksSeeUncommittedEdgesAndKeepOwnerAndTypeBoundaries(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, query := range []string{
		`CREATE TABLE blackboard_v2_session_relationships (session_id TEXT, from_key TEXT, relation TEXT, to_key TEXT)`,
		`CREATE TABLE blackboard_v2_session_records (session_id TEXT, key TEXT, type TEXT)`,
		`INSERT INTO blackboard_v2_session_records VALUES ('s','a','fact'),('s','b','fact'),('s','c','fact'),('s','entity','entity')`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, query := range []string{
		`INSERT INTO blackboard_v2_session_relationships VALUES ('s','b','supports','c'),('s','c','supports','a'),('s','b','part_of','entity'),('other','entity','part_of','a')`,
	} {
		if _, err := tx.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	for _, tc := range []struct {
		relation, from, to string
		want               bool
	}{
		{"supports", "a", "b", true}, {"supports", "b", "a", false},
		{"part_of", "a", "b", false}, {"supports", "a", "a", true},
	} {
		got, err := sessionRelationshipWouldCycle(ctx, tx, "s", tc.relation, tc.from, tc.to)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: got %v, %v", tc, got, err)
		}
	}
	if got, err := sessionFactSupportsWouldCycle(ctx, tx, "s", "a", "b"); err != nil || !got {
		t.Fatalf("Fact cycle: %v, %v", got, err)
	}
	if _, err := tx.Exec(`INSERT INTO blackboard_v2_session_relationships VALUES ('s','b','supports','entity'),('s','entity','supports','a'),('s','c','supports','b')`); err != nil {
		t.Fatal(err)
	}
	// The existing b/c cycle must terminate. Entity hops must not join the Fact graph.
	if got, err := sessionFactSupportsWouldCycle(ctx, tx, "s", "entity", "b"); err != nil || got {
		t.Fatalf("Fact type boundary: %v, %v", got, err)
	}
	if got, err := sessionRelationshipWouldCycle(ctx, tx, "s", "supports", "missing", "b"); err != nil || got {
		t.Fatalf("retained cycle: %v, %v", got, err)
	}
}
