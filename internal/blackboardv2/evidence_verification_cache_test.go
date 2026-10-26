package blackboardv2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVerifiedEvidenceCacheKeepsIntegrityBoundaries(t *testing.T) {
	root := t.TempDir()
	s := NewServiceWithEvidence(nil, EvidenceConfig{ArtifactRoot: root})
	content := []byte("verified content")
	digest := sha256.Sum256(content)
	path := filepath.Join(root, "payload.bin")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	wantDigest := hex.EncodeToString(digest[:])
	for i := 0; i < 2; i++ {
		if valid, err := s.verifyManagedEvidencePayload("payload.bin", wantDigest, int64(len(content))); err != nil || !valid {
			t.Fatalf("verify %d: %v %v", i, valid, err)
		}
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "linux" && !s.evidenceVerified.matches("payload.bin", wantDigest, int64(len(content)), before) {
		t.Fatal("verified file was not cached")
	}
	// Size and mtime alone are insufficient: a writer can restore mtime.
	changed := []byte("tampered content")
	if len(changed) != len(content) {
		t.Fatal("test payload sizes differ")
	}
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	if valid, err := s.verifyManagedEvidencePayload("payload.bin", wantDigest, int64(len(content))); err == nil || valid {
		t.Fatalf("same-size tamper accepted: %v %v", valid, err)
	}
}

func TestVerifiedEvidenceCacheRejectsReplacementAndSymlink(t *testing.T) {
	root := t.TempDir()
	s := NewServiceWithEvidence(nil, EvidenceConfig{ArtifactRoot: root})
	content := []byte("original")
	digest := sha256.Sum256(content)
	path := filepath.Join(root, "payload.bin")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	encoded := hex.EncodeToString(digest[:])
	if ok, err := s.verifyManagedEvidencePayload("payload.bin", encoded, int64(len(content))); err != nil || !ok {
		t.Fatal(ok, err)
	}
	other := filepath.Join(root, "replacement.bin")
	if err := os.WriteFile(other, []byte("modified"), 0600); err != nil {
		t.Fatal(err)
	}
	// Rename replaces the inode at the cached path.
	if err := os.Rename(other, path); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.verifyManagedEvidencePayload("payload.bin", encoded, int64(len(content))); err == nil || ok {
		t.Fatal("replacement accepted", ok, err)
	}
	// A warm cache entry must not bypass the secure path opener.
	cached := filepath.Join(root, "cached.bin")
	if err := os.WriteFile(cached, content, 0600); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.verifyManagedEvidencePayload("cached.bin", encoded, int64(len(content))); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if err := os.Rename(cached, filepath.Join(root, "original.bin")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("original.bin", cached); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if ok, err := s.verifyManagedEvidencePayload("cached.bin", encoded, int64(len(content))); err == nil || ok {
		t.Fatal("cached symlink accepted", ok, err)
	}
	if err := os.Symlink("payload.bin", filepath.Join(root, "link.bin")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if ok, err := s.verifyManagedEvidencePayload("link.bin", encoded, int64(len(content))); err == nil || ok {
		t.Fatal("symlink accepted", ok, err)
	}
}

func TestVerifiedEvidenceCacheIsBoundedAndDoesNotCacheChangingFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "payload.bin")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := evidenceChangeTime(before); !ok {
		t.Skip("kernel change timestamp unavailable")
	}
	cache := evidenceVerificationCache{}
	for i := 0; i < maxVerifiedEvidenceFiles+5; i++ {
		cache.remember(fmt.Sprint(i), "digest", 8, before, before)
	}
	if len(cache.entries) != maxVerifiedEvidenceFiles || len(cache.order) != maxVerifiedEvidenceFiles {
		t.Fatalf("cache size = %d/%d", len(cache.entries), len(cache.order))
	}
	if cache.matches("0", "digest", 8, before) {
		t.Fatal("oldest entry was not evicted")
	}
	if err := os.WriteFile(path, []byte("modified"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	cache.remember("changed", "digest", 8, before, after)
	if _, ok := cache.entries["changed"]; ok {
		t.Fatal("file changed while hashing was cached")
	}
}

func BenchmarkVerifiedEvidencePayload(b *testing.B) {
	for _, warm := range []bool{false, true} {
		b.Run(fmt.Sprintf("warm=%v", warm), func(b *testing.B) {
			root := b.TempDir()
			s := NewServiceWithEvidence(nil, EvidenceConfig{ArtifactRoot: root})
			content := make([]byte, 1024*1024)
			digest := sha256.Sum256(content)
			encoded := hex.EncodeToString(digest[:])
			if err := os.WriteFile(filepath.Join(root, "payload.bin"), content, 0600); err != nil {
				b.Fatal(err)
			}
			if _, err := s.verifyManagedEvidencePayload("payload.bin", encoded, int64(len(content))); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if !warm {
					s.evidenceVerified = evidenceVerificationCache{}
				}
				if valid, err := s.verifyManagedEvidencePayload("payload.bin", encoded, int64(len(content))); err != nil || !valid {
					b.Fatal(valid, err)
				}
			}
		})
	}
}
