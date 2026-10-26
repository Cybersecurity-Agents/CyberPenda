package blackboardv2

import (
	"context"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"testing"
)

func cycleQueryDB(tb testing.TB) *sql.DB {
	tb.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { db.Close() })
	for _, stmt := range []string{
		`CREATE TABLE blackboard_v2_relationships(project_id TEXT, from_key TEXT, relation TEXT, to_key TEXT, PRIMARY KEY(project_id,from_key,relation,to_key))`,
		`CREATE TABLE blackboard_v2_records(project_id TEXT, key TEXT, type TEXT, PRIMARY KEY(project_id,key))`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			tb.Fatal(err)
		}
	}
	return db
}

func TestCycleQueriesUseTransactionAndProjectBoundaries(t *testing.T) {
	db := cycleQueryDB(t)
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, edge := range [][4]string{{"p", "a", "part_of", "b"}, {"p", "b", "part_of", "a"}, {"other", "c", "part_of", "a"}, {"p", "a", "supports", "b"}, {"p", "b", "supports", "c"}} {
		if _, err := tx.Exec(`INSERT INTO blackboard_v2_relationships VALUES(?,?,?,?)`, edge[0], edge[1], edge[2], edge[3]); err != nil {
			t.Fatal(err)
		}
	}
	for _, record := range [][2]string{{"a", "fact"}, {"b", "fact"}, {"c", "finding"}} {
		if _, err := tx.Exec(`INSERT INTO blackboard_v2_records VALUES('p',?,?)`, record[0], record[1]); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		from, to string
		want     bool
	}{{"b", "a", true}, {"c", "a", false}, {"a", "c", false}, {"a", "a", true}} {
		got, err := relationshipWouldCycle(ctx, tx, "p", "part_of", test.from, test.to)
		if err != nil || got != test.want {
			t.Fatalf("part_of %s -> %s = %v,%v; want %v", test.from, test.to, got, err, test.want)
		}
	}
	if got, err := factSupportsWouldCycle(ctx, tx, "p", "b", "a"); err != nil || !got {
		t.Fatalf("Fact path = %v,%v", got, err)
	}
	if got, err := factSupportsWouldCycle(ctx, tx, "p", "c", "a"); err != nil || got {
		t.Fatalf("Finding endpoint entered Fact path = %v,%v", got, err)
	}
}

func BenchmarkProjectRelationshipCycleChain(b *testing.B) {
	db := cycleQueryDB(b)
	tx, err := db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Rollback()
	for i := 0; i < 256; i++ {
		if _, err := tx.Exec(`INSERT INTO blackboard_v2_relationships VALUES('p',?,'part_of',?)`, fmt.Sprint(i), fmt.Sprint(i+1)); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, err := relationshipWouldCycle(context.Background(), tx, "p", "part_of", "256", "0")
		if err != nil || !got {
			b.Fatalf("cycle=%v,%v", got, err)
		}
	}
}
