package blackboardv2

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"pentest/internal/project"
	"pentest/internal/store"
)

// Keep the affected-key filter indexable as Projects gain unrelated Findings
// and Solutions. Avoid replacing it with a Go filter after a Project scan.
func TestConclusionValidationAffectedKeysUsePrimaryKeyProbes(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, typ := range []string{"finding", "solution"} {
		rows, err := db.Query(`EXPLAIN QUERY PLAN SELECT key,record_json FROM blackboard_v2_records
   WHERE project_id=? AND type=? AND key IN (SELECT value FROM json_each(?)) ORDER BY key`, "project", typ, `["selected"]`)
		if err != nil {
			t.Fatal(err)
		}
		indexed := false
		details := []string{}
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			details = append(details, detail)
			indexed = indexed || strings.Contains(detail, "project_id=? AND key=?")
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		rows.Close()
		if !indexed {
			t.Fatalf("%s validation does not probe selected primary keys: %v", typ, details)
		}
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
