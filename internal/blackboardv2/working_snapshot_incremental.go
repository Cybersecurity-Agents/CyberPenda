package blackboardv2

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
)

// Reuse only a snapshot that exactly matches the transaction's batch base.
// Other writers can advance the Project between this Continuation's batches;
// that case must use a full projection. Encoding the complete document remains
// O(snapshot size), but unchanged full record payloads are not loaded/decoded.
func (s *Service) projectWorkingSnapshotTx(ctx context.Context, tx *sql.Tx, projectID string, previousRevision int, previousBytes []byte, changes []workingSnapshotChanges) (RuntimeSnapshotProjection, error) {
	if len(changes) != 1 || changes[0].baseRevision != previousRevision || len(changes[0].records) > 500 {
		return s.ProjectRuntimeSnapshotTx(ctx, tx, projectID)
	}
	var previous RuntimeSnapshot
	if json.Unmarshal(previousBytes, &previous) != nil || previous.Schema != snapshotSchema || previous.Semantics != snapshotSemantics || previous.Revision != previousRevision {
		return s.ProjectRuntimeSnapshotTx(ctx, tx, projectID)
	}
	keys := make([]string, 0, len(changes[0].records))
	for key := range changes[0].records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	patch, err := s.runtimeSnapshotForKeysTx(ctx, tx, projectID, keys)
	if err != nil {
		return RuntimeSnapshotProjection{}, err
	}
	previous.Revision = patch.Revision
	previous.Work.Objectives = mergeWorkingSnapshotRecords(previous.Work.Objectives, patch.Work.Objectives, keys)
	previous.Work.Attempts = mergeWorkingSnapshotRecords(previous.Work.Attempts, patch.Work.Attempts, keys)
	previous.Knowledge.Entities = mergeWorkingSnapshotRecords(previous.Knowledge.Entities, patch.Knowledge.Entities, keys)
	previous.Knowledge.Facts = mergeWorkingSnapshotRecords(previous.Knowledge.Facts, patch.Knowledge.Facts, keys)
	previous.Knowledge.Findings = mergeWorkingSnapshotRecords(previous.Knowledge.Findings, patch.Knowledge.Findings, keys)
	previous.Knowledge.Solutions = mergeWorkingSnapshotRecords(previous.Knowledge.Solutions, patch.Knowledge.Solutions, keys)
	previous.Knowledge.Evidence = mergeWorkingSnapshotRecords(previous.Knowledge.Evidence, patch.Knowledge.Evidence, keys)
	previous.Relations = patch.Relations
	return projectRuntimeSnapshot(previous)
}

func mergeWorkingSnapshotRecords[T any](previous, patch map[string]T, keys []string) map[string]T {
	for _, key := range keys {
		delete(previous, key)
	}
	if previous == nil && len(patch) > 0 {
		previous = make(map[string]T, len(patch))
	}
	for key, value := range patch {
		previous[key] = value
	}
	return previous
}
