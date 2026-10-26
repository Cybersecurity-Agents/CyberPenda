package fgs

import (
	"context"
	"encoding/json"
	"strings"
)

// readUpdateGraph loads operation dependencies and their Goal ancestry. The
// accepted graph is valid before this transaction. Ordinary updates only need
// this read set to check changed nodes against already-completed ancestors.
// Goal completion also checks untouched descendants, so it keeps the full read.
func readUpdateGraph(ctx context.Context, q queryer, kind, id string, operations []Operation) (Graph, error) {
	for _, op := range operations {
		if op.Op == "goal.transition" && op.To == "done" {
			return readGraph(ctx, q, kind, id)
		}
	}
	g := Graph{Nodes: []Node{}, Edges: []Edge{}}
	if err := q.QueryRowContext(ctx, `SELECT revision FROM fgs_boards WHERE kind=? AND owner_id=?`, kind, id).Scan(&g.Revision); err != nil {
		return g, err
	}
	pending := []string{}
	seen := map[string]bool{}
	add := func(key string) {
		if key != "" && !seen[key] {
			seen[key] = true
			pending = append(pending, key)
		}
	}
	for _, op := range operations {
		for _, key := range []string{op.Key, op.Goal, op.Step, op.ParentGoal, op.Corrects} {
			add(key)
		}
		for _, keys := range [][]string{op.Inputs, op.Outputs, op.Facts, op.After} {
			for _, key := range keys {
				add(key)
			}
		}
	}
	for len(pending) > 0 {
		count := len(pending)
		if count > 900 {
			count = 900
		}
		batch := pending[:count]
		pending = pending[count:]
		args := []any{kind, id}
		for _, key := range batch {
			args = append(args, key)
		}
		marks := strings.TrimSuffix(strings.Repeat("?,", count), ",")
		rows, err := q.QueryContext(ctx, `SELECT body_json FROM fgs_nodes WHERE board_kind=? AND board_id=? AND node_key IN (`+marks+`)`, args...)
		if err != nil {
			return g, err
		}
		for rows.Next() {
			var raw string
			if err = rows.Scan(&raw); err != nil {
				rows.Close()
				return g, err
			}
			var n Node
			if err = json.Unmarshal([]byte(raw), &n); err != nil {
				rows.Close()
				return g, err
			}
			g.Nodes = append(g.Nodes, n)
			add(n.Goal)
			add(n.Step)
			add(n.ParentGoal)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return g, err
		}
	}
	return g, nil
}
