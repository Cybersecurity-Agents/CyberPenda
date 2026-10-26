package fgs

import (
	"context"
	"encoding/json"
	"path/filepath"
	"pentest/internal/store"
	"testing"
)

func TestUpdateGraphLoadsReferencesAndAncestorsWithoutUnrelatedNodes(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "graph.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`INSERT INTO fgs_boards(kind,owner_id,revision) VALUES('session','owner',7)`); err != nil {
		t.Fatal(err)
	}
	for _, n := range []Node{{Key: "goal:root", Type: "goal", State: "open"}, {Key: "goal:child", Type: "goal", State: "open", ParentGoal: "goal:root"}, {Key: "step:one", Type: "step", Goal: "goal:child", State: "open"}, {Key: "goal:unrelated", Type: "goal", State: "open"}} {
		raw, _ := json.Marshal(n)
		if _, err = db.Exec(`INSERT INTO fgs_nodes VALUES('session','owner',?,?)`, n.Key, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	g, err := readUpdateGraph(context.Background(), db, "session", "owner", []Operation{{Op: "fact.append", Key: "fact:new", Step: "step:one"}})
	if err != nil {
		t.Fatal(err)
	}
	if g.Revision != 7 || len(g.Nodes) != 3 {
		t.Fatalf("read set = %+v", g)
	}
	for _, n := range g.Nodes {
		if n.Key == "goal:unrelated" {
			t.Fatal("loaded unrelated Goal")
		}
	}
	g, err = readUpdateGraph(context.Background(), db, "session", "owner", []Operation{{Op: "goal.transition", Key: "goal:root", To: "done"}})
	if err != nil || len(g.Nodes) != 4 {
		t.Fatalf("completion read set = %+v, %v", g, err)
	}
}
