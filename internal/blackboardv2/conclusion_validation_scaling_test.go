package blackboardv2

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"pentest/internal/project"
	"pentest/internal/store"
)

// Exercise the real validators, not a copied SQL statement. Both an accidental
// full-table decode and a validator that skips affected rows must fail here.
func TestSelectedConclusionValidationIsolatesLargeUnrelatedCorpus(t *testing.T) {
	for _, typ := range []string{"finding", "solution"} {
		t.Run(typ, func(t *testing.T) {
			db, err := store.Open("")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			kind := project.KindPentest
			var record any = findingOutputRecord{Status: "unconfirmed", Title: "Selected Finding", CVSSPending: true}
			validate := validateConfirmedFindings
			if typ == "solution" {
				kind = project.KindCTFChallenge
				record = SolutionRecord{Status: "candidate", Kind: "flag", Summary: "Selected Solution", Value: "flag"}
				validate = validateVerifiedSolutions
			}
			proj, err := project.NewService(db).CreateWithKind("Selected validation", "", kind, project.Scope{}, project.Defaults{})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			insert := func(key, raw string) {
				t.Helper()
				if _, err := tx.Exec(`INSERT INTO blackboard_v2_records(project_id,key,type,version,record_json,created_at,updated_at) VALUES(?,?,?,1,?,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`, proj.ID, key, typ, raw); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 1000; i++ {
				insert(fmt.Sprintf("%s:unrelated-%04d", typ, i), "invalid unrelated JSON")
			}
			selected := []string{typ + ":selected-a", typ + ":selected-b"}
			for _, key := range selected {
				insert(key, string(encoded))
			}
			if err := validate(context.Background(), tx, proj.ID, selected); err != nil {
				t.Fatalf("selected validation decoded an unrelated row: %v", err)
			}
			if err := validate(context.Background(), tx, proj.ID, nil); err == nil {
				t.Fatal("full validation skipped unrelated corrupt rows")
			}
			for _, key := range selected {
				if _, err := tx.Exec(`UPDATE blackboard_v2_records SET record_json='invalid selected JSON' WHERE project_id=? AND key=?`, proj.ID, key); err != nil {
					t.Fatal(err)
				}
				if err := validate(context.Background(), tx, proj.ID, selected); err == nil {
					t.Fatalf("selected corrupt row %s escaped validation", key)
				}
				if _, err := tx.Exec(`UPDATE blackboard_v2_records SET record_json=? WHERE project_id=? AND key=?`, string(encoded), proj.ID, key); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkConclusionValidationAffectedKeys(b *testing.B) {
	for _, typ := range []string{"finding", "solution"} {
		for _, count := range []int{100, 10000} {
			b.Run(fmt.Sprintf("%s/records=%d", typ, count), func(b *testing.B) {
				db, err := store.Open("")
				if err != nil {
					b.Fatal(err)
				}
				defer db.Close()
				kind := project.KindPentest
				var record any = findingOutputRecord{Status: "unconfirmed", Title: "Fixture Finding", CVSSPending: true}
				validate := validateConfirmedFindings
				if typ == "solution" {
					kind = project.KindCTFChallenge
					record = SolutionRecord{Status: "candidate", Kind: "flag", Summary: "Fixture Solution", Value: "flag"}
					validate = validateVerifiedSolutions
				}
				proj, err := project.NewService(db).CreateWithKind("Validation scaling", "", kind, project.Scope{}, project.Defaults{})
				if err != nil {
					b.Fatal(err)
				}
				encoded, err := json.Marshal(record)
				if err != nil {
					b.Fatal(err)
				}
				tx, err := db.Begin()
				if err != nil {
					b.Fatal(err)
				}
				stmt, err := tx.Prepare(`INSERT INTO blackboard_v2_records(project_id,key,type,version,record_json,created_at,updated_at) VALUES(?,?,?,1,?,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`)
				if err != nil {
					tx.Rollback()
					b.Fatal(err)
				}
				for i := 0; i < count; i++ {
					if _, err := stmt.Exec(proj.ID, fmt.Sprintf("%s:%05d", typ, i), typ, string(encoded)); err != nil {
						stmt.Close()
						tx.Rollback()
						b.Fatal(err)
					}
				}
				stmt.Close()
				if err := tx.Commit(); err != nil {
					b.Fatal(err)
				}
				for _, targeted := range []bool{false, true} {
					b.Run(fmt.Sprintf("targeted=%v", targeted), func(b *testing.B) {
						tx, err := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
						if err != nil {
							b.Fatal(err)
						}
						defer tx.Rollback()
						var keys []string
						if targeted {
							keys = []string{fmt.Sprintf("%s:%05d", typ, 0)}
						}
						b.ReportAllocs()
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							if err := validate(context.Background(), tx, proj.ID, keys); err != nil {
								b.Fatal(err)
							}
						}
					})
				}
			})
		}
	}
}
