package blackboardv2

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"pentest/internal/project"
	"pentest/internal/store"
)

func TestConfirmedFactBasisSkipsUnneededEvidenceIO(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	p, err := project.NewService(db).Create("Fact basis", "", project.Scope{}, project.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	s := NewServiceWithEvidence(db, EvidenceConfig{ArtifactRoot: filepath.Join(t.TempDir(), "missing-artifact-root")})
	digest := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	internal, err := plannedEvidenceInternalPath(p.ID, digest, "proof.txt")
	if err != nil {
		t.Fatal(err)
	}
	managed, err := semanticEvidencePath(p.ID, internal, digest)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		key, typ string
		record   any
	}{
		{"fact:target", "fact", FactRecord{Category: "test", Summary: "Target", Confidence: "confirmed", ScopeStatus: "unknown"}},
		{"fact:support", "fact", FactRecord{Category: "test", Summary: "Support", Confidence: "confirmed", ScopeStatus: "unknown"}},
		{"evidence:proof", "evidence", EvidenceRecord{Status: "available", ArtifactType: "text", Summary: "Proof", ManagedPath: managed, SHA256: digest, Size: 1024 * 1024 * 1024}},
	} {
		encoded, err := json.Marshal(item.record)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO blackboard_v2_records(project_id,key,type,version,record_json,created_at,updated_at) VALUES(?,?,?,1,?,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`, p.ID, item.key, item.typ, string(encoded)); err != nil {
			t.Fatal(err)
		}
	}
	for _, edge := range []struct{ from, relation string }{{"fact:support", "supports"}, {"evidence:proof", "evidences"}} {
		if _, err := db.Exec(`INSERT INTO blackboard_v2_relationships(project_id,from_key,relation,to_key,version,reason,created_at,updated_at) VALUES(?,?,?,'fact:target',1,'','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`, p.ID, edge.from, edge.relation); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	// The missing Artifact Root is a filesystem-read sentinel. A confirmed Fact
	// already supplies the required basis; no Evidence file needs to be opened.
	if valid, err := s.factHasDurableBasis(context.Background(), tx, p.ID, "fact:target"); err != nil || !valid {
		t.Fatalf("confirmed Fact basis performed unneeded Evidence I/O: %v %v", valid, err)
	}
	if _, err := tx.Exec(`DELETE FROM blackboard_v2_relationships WHERE project_id=? AND relation='supports'`, p.ID); err != nil {
		t.Fatal(err)
	}
	// Without an alternate basis, the existing full Evidence verifier must run.
	if valid, err := s.factHasDurableBasis(context.Background(), tx, p.ID, "fact:target"); err == nil || valid {
		t.Fatalf("Evidence-only basis skipped integrity verification: %v %v", valid, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	// The basis is an OR contract. A valid supporting Fact does not make
	// unavailable Evidence healthy; health still reports it independently.
	s.evidenceConfig.ArtifactRoot = t.TempDir()
	health, err := s.ProjectSemanticHealth(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	reported := false
	for _, anomaly := range health.Anomalies {
		reported = reported || (anomaly.Code == "evidence_integrity" && anomaly.SubjectKey == "evidence:proof")
	}
	if !reported {
		t.Fatal("health did not report unavailable Evidence")
	}
}
