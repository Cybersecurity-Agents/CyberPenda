package blackboardv2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pentest/internal/project"
	"pentest/internal/store"
)

func newApplyEvidenceFixture(t *testing.T) (*Service, string, string, EvidenceRecord, string) {
	t.Helper()
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	p, err := project.NewService(db).Create("Evidence preflight", "", project.Scope{}, project.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	s := NewServiceWithEvidence(db, EvidenceConfig{ArtifactRoot: root})
	data := []byte("verified content")
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	internal, err := plannedEvidenceInternalPath(p.ID, digest, "proof.txt")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, internal)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	managed, err := semanticEvidencePath(p.ID, internal, digest)
	if err != nil {
		t.Fatal(err)
	}
	record := EvidenceRecord{Status: "available", ArtifactType: "text", Summary: "Proof", ManagedPath: managed, SHA256: digest, Size: int64(len(data))}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	key := "evidence:proof"
	if _, err := db.Exec(`INSERT INTO blackboard_v2_records(project_id,key,type,version,record_json,created_at,updated_at) VALUES(?,?,'evidence',1,?,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`, p.ID, key, string(raw)); err != nil {
		t.Fatal(err)
	}
	return s, p.ID, key, record, path
}

func TestApplyEvidencePreflightReadsBytesAgainForEachRequest(t *testing.T) {
	s, pid, key, record, path := newApplyEvidenceFixture(t)
	batch := ChangeBatch{Changes: []Change{{Op: "update", Key: key}}}
	ctx := s.preflightApplyEvidence(context.Background(), pid, batch)
	if ok, err := s.applyEvidenceIntegrityValid(ctx, pid, key, 1, record); err != nil || !ok {
		t.Fatal(ok, err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tampered content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	// A second Apply cannot reuse the first request's verified digest, even
	// when file size and mtime are unchanged.
	next := s.preflightApplyEvidence(context.Background(), pid, batch)
	if ok, err := s.applyEvidenceIntegrityValid(next, pid, key, 1, record); err != nil || ok {
		t.Fatal("second request accepted altered bytes", ok, err)
	}
	// A changed semantic version cannot use the old request's preflight result.
	if ok, err := s.applyEvidenceIntegrityValid(ctx, pid, key, 2, record); err != nil || ok {
		t.Fatal("new Evidence version reused old verification", ok, err)
	}
}

func TestApplyEvidencePreflightDoesNotHoldWriterLockOrTransaction(t *testing.T) {
	s, pid, key, _, _ := newApplyEvidenceFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	done := make(chan context.Context, 1)
	go func() {
		done <- s.preflightApplyEvidence(ctx, pid, ChangeBatch{Changes: []Change{{Op: "update", Key: key}}})
	}()
	select {
	case checked := <-done:
		result, ok := checked.Value(applyEvidenceContextKey{}).(applyEvidenceChecks)
		if !ok || !result.byKey[key].valid {
			t.Fatal("preflight did not verify retained bytes")
		}
	case <-ctx.Done():
		t.Fatal("preflight waited on writer mutex")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("preflight retained a database transaction: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
}
