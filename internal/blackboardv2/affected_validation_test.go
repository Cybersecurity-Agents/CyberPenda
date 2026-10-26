package blackboardv2_test

import (
	"context"
	"path/filepath"
	"pentest/internal/blackboardv2"
	"pentest/internal/project"
	"pentest/internal/store"
	"strings"
	"testing"
)

func TestApplyValidationReadsOnlyAffectedConclusionRows(t *testing.T) {
	for _, kind := range []string{project.KindPentest, project.KindCTFChallenge} {
		t.Run(string(kind), func(t *testing.T) {
			db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			proj, err := project.NewService(db).CreateWithKind("P", "", kind, project.Scope{}, project.Defaults{})
			if err != nil {
				t.Fatal(err)
			}
			service := blackboardv2.NewService(db)
			typ, key := "finding", "finding:untouched"
			var record any = blackboardv2.FindingRecord{Status: "unconfirmed", Title: "Untouched"}
			if kind == project.KindCTFChallenge {
				typ, key = "solution", "solution:untouched"
				record = blackboardv2.SolutionRecord{Status: "candidate", Kind: "flag", Summary: "Untouched", Value: "flag"}
			}
			_, err = service.Apply(context.Background(), proj.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "seed", Changes: []blackboardv2.Change{{Op: "create", Key: key, Type: typ, Record: record}}})
			if err != nil {
				t.Fatal(err)
			}
			// A malformed unrelated row is a read sentinel. Report validation must still
			// inspect it, but an unrelated change must not decode the whole conclusion table.
			if _, err := db.Exec(`UPDATE blackboard_v2_records SET record_json='bad JSON' WHERE project_id=? AND key=?`, proj.ID, key); err != nil {
				t.Fatal(err)
			}
			_, err = service.Apply(context.Background(), proj.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "unrelated", Changes: []blackboardv2.Change{{Op: "create", Key: "entity:other", Type: "entity", Record: blackboardv2.EntityRecord{Status: "active", Kind: "host", Name: "Other", ScopeStatus: "in_scope"}}}})
			if err != nil {
				t.Fatalf("unrelated conclusion was decoded: %v", err)
			}
			if kind == project.KindPentest {
				if _, err := service.PentestReport(context.Background(), proj.ID); err == nil {
					t.Fatal("report skipped whole-project validation")
				}
			}
		})
	}
}

func TestAffectedFindingSupportChangesValidateAndRollback(t *testing.T) {
	for _, operation := range []string{"unrelate", "demote", "supersede", "replace_support"} {
		t.Run(operation, func(t *testing.T) {
			db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			proj, err := project.NewService(db).Create("P", "", project.Scope{}, project.Defaults{})
			if err != nil {
				t.Fatal(err)
			}
			service := blackboardv2.NewService(db)
			ctx := context.Background()
			seed := []blackboardv2.Change{
				{Op: "create", Key: "fact:proof", Type: "fact", Record: blackboardv2.FactRecord{Category: "validation", Summary: "Reproduced", Confidence: "confirmed", ScopeStatus: "in_scope"}},
				{Op: "create", Key: "fact:replacement", Type: "fact", Record: blackboardv2.FactRecord{Category: "validation", Summary: "Other reproduction", Confidence: "confirmed", ScopeStatus: "in_scope"}},
				{Op: "create", Key: "finding:confirmed", Type: "finding", Record: blackboardv2.FindingRecord{Status: "confirmed", Title: "Issue", Target: "https://example.test", Proof: "Reproduced", Impact: "Access", Recommendation: "Fix", CVSSVersion: "4.0", CVSSVector: criticalCVSS40}},
				{Op: "relate", From: "fact:proof", Relation: "supports", To: "finding:confirmed"},
			}
			if _, err := service.Apply(ctx, proj.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "seed", Changes: seed}); err != nil {
				t.Fatal(err)
			}
			changes := []blackboardv2.Change{{Op: "create", Key: "entity:marker", Type: "entity", Record: blackboardv2.EntityRecord{Status: "active", Kind: "host", Name: "Marker", ScopeStatus: "in_scope"}}}
			switch operation {
			case "unrelate":
				changes = append(changes, blackboardv2.Change{Op: "unrelate", From: "fact:proof", Relation: "supports", To: "finding:confirmed", Version: 1})
			case "demote":
				changes = append(changes, blackboardv2.Change{Op: "transition", Key: "fact:proof", Version: 1, Status: "tentative"})
			case "supersede":
				changes = append(changes, blackboardv2.Change{Op: "supersede", Replaced: "fact:proof", ReplacedVersion: 1, Replacement: "fact:replacement", ReplacementVersion: 1})
			case "replace_support":
				changes = append(changes, blackboardv2.Change{Op: "unrelate", From: "fact:proof", Relation: "supports", To: "finding:confirmed", Version: 1}, blackboardv2.Change{Op: "relate", From: "fact:replacement", Relation: "supports", To: "finding:confirmed"})
			}
			_, err = service.Apply(ctx, proj.ID, blackboardv2.ChangeBatch{Schema: "semantic-change-batch/v2", IdempotencyKey: "change", Changes: changes})
			if operation == "replace_support" {
				if err != nil {
					t.Fatalf("support restored in same batch: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "confirmed Finding requires") {
				t.Fatalf("support invariant not validated: %v", err)
			}
			if _, err := service.ReadCurrent(ctx, proj.ID, "entity:marker"); !isSemanticCode(err, "not_found") {
				t.Fatalf("marker survived rejected batch: %v", err)
			}
			finding, err := service.ReadCurrent(ctx, proj.ID, "finding:confirmed")
			if err != nil {
				t.Fatal(err)
			}
			if len(finding.Relationships) != 1 {
				t.Fatalf("rollback lost support: %#v", finding.Relationships)
			}
		})
	}
}
