package blackboardv2

import (
	"context"
	"encoding/json"
	"fmt"
	"pentest/internal/project"
	"pentest/internal/store"
	"strings"
	"testing"
)

// Full stored Fact bodies are not part of the compact model-visible Snapshot.
// This compares repeated full-record decoding with a one-key projection.
func BenchmarkWorkingSnapshotChangedRecord(b *testing.B) {
	db, err := store.Open("")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	p, err := project.NewService(db).Create("Snapshot benchmark", "", project.Scope{}, project.Defaults{})
	if err != nil {
		b.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO blackboard_v2_project_state(project_id,revision) VALUES (?,500)`, p.ID); err != nil {
		b.Fatal(err)
	}
	raw, err := json.Marshal(FactRecord{Category: "test", Summary: "Reference fact", Body: strings.Repeat("x", 4096), Confidence: "tentative", ScopeStatus: "unknown"})
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 500; i++ {
		if _, err := tx.Exec(`INSERT INTO blackboard_v2_records(project_id,key,type,version,record_json,created_at,updated_at) VALUES (?,?,'fact',1,?,'','')`, p.ID, fmt.Sprintf("fact:%d", i), string(raw)); err != nil {
			b.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	service := NewService(db)
	ctx := context.Background()
	prior, err := service.ProjectRuntimeSnapshot(ctx, p.ID)
	if err != nil {
		b.Fatal(err)
	}
	tx, err = db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE blackboard_v2_records SET version=2 WHERE project_id=? AND key='fact:0'`, p.ID); err != nil {
		b.Fatal(err)
	}
	if _, err := tx.Exec(`UPDATE blackboard_v2_project_state SET revision=501 WHERE project_id=?`, p.ID); err != nil {
		b.Fatal(err)
	}
	changes := []workingSnapshotChanges{{baseRevision: 500, records: map[string]int{"fact:0": 2}}}
	for _, mode := range []string{"full", "changed_record"} {
		b.Run(mode, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if mode == "full" {
					_, err = service.ProjectRuntimeSnapshotTx(ctx, tx, p.ID)
				} else {
					_, err = service.projectWorkingSnapshotTx(ctx, tx, p.ID, 500, prior.Bytes, changes)
				}
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
