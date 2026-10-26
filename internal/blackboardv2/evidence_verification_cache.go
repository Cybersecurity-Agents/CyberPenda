package blackboardv2

import (
	"os"
	"sync"
)

const maxVerifiedEvidenceFiles = 256

type verifiedEvidenceFile struct {
	info   os.FileInfo
	digest string
	size   int64
}

type evidenceVerificationCache struct {
	mu      sync.Mutex
	entries map[string]verifiedEvidenceFile
	order   []string
}

func sameVerifiedEvidenceFile(before, after os.FileInfo) bool {
	beforeSec, beforeNS, beforeOK := evidenceChangeTime(before)
	afterSec, afterNS, afterOK := evidenceChangeTime(after)
	return beforeOK && afterOK && os.SameFile(before, after) &&
		before.Mode() == after.Mode() && before.Size() == after.Size() &&
		before.ModTime().Equal(after.ModTime()) && beforeSec == afterSec && beforeNS == afterNS
}

func (cache *evidenceVerificationCache) matches(path, digest string, size int64, info os.FileInfo) bool {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	prior, ok := cache.entries[path]
	return ok && prior.digest == digest && prior.size == size && sameVerifiedEvidenceFile(prior.info, info)
}

func (cache *evidenceVerificationCache) remember(path, digest string, size int64, before, after os.FileInfo) {
	// A successful hash can be reused only for the same unchanged inode.
	// Platforms without a kernel change timestamp keep the full-hash path.
	if !sameVerifiedEvidenceFile(before, after) {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.entries == nil {
		cache.entries = make(map[string]verifiedEvidenceFile)
	}
	if _, ok := cache.entries[path]; !ok {
		if len(cache.order) == maxVerifiedEvidenceFiles {
			delete(cache.entries, cache.order[0])
			copy(cache.order, cache.order[1:])
			cache.order = cache.order[:len(cache.order)-1]
		}
		cache.order = append(cache.order, path)
	}
	cache.entries[path] = verifiedEvidenceFile{info: after, digest: digest, size: size}
}
