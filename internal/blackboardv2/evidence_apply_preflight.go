package blackboardv2

import (
	"context"
	"encoding/json"
)

type applyEvidenceContextKey struct{}

type applyEvidenceCheck struct {
	version int
	record  EvidenceRecord
	valid   bool
}

type applyEvidenceChecks struct {
	service   *Service
	projectID string
	byKey     map[string]applyEvidenceCheck
}

// preflightApplyEvidence verifies retained bytes for potentially affected Fact
// bases. Queries finish before file I/O, so neither the writer mutex nor a
// database transaction is held while hashing. Missing candidates and concurrent
// semantic changes use the ordinary in-transaction verifier as a fallback.
func (s *Service) preflightApplyEvidence(ctx context.Context, projectID string, batch ChangeBatch) context.Context {
	keys := make([]string, 0, len(batch.Changes)*3)
	for _, change := range batch.Changes {
		for _, key := range []string{change.Key, change.From, change.To, change.Source, change.Canonical, change.Replaced} {
			if key != "" {
				keys = append(keys, key)
			}
		}
	}
	if len(keys) == 0 {
		return ctx
	}
	raw, err := json.Marshal(keys)
	if err != nil {
		return ctx
	}
	rows, err := s.db.QueryContext(ctx, `WITH seeds AS (SELECT value AS key FROM json_each(?)),
 affected AS (
  SELECT key FROM seeds
  UNION SELECT rel.to_key FROM blackboard_v2_relationships AS rel
   WHERE rel.project_id=? AND rel.from_key IN (SELECT key FROM seeds)
    AND rel.relation IN ('supports','evidences')
 )
 SELECT evidence.key,evidence.version,evidence.record_json
 FROM blackboard_v2_records AS evidence
 WHERE evidence.project_id=? AND evidence.type='evidence' AND evidence.key IN (
  SELECT key FROM seeds
  UNION SELECT rel.from_key FROM blackboard_v2_relationships AS rel
   WHERE rel.project_id=? AND rel.relation='evidences' AND rel.to_key IN (SELECT key FROM affected)
 )`, string(raw), projectID, projectID, projectID)
	if err != nil {
		return ctx
	}
	type candidate struct {
		key     string
		version int
		record  EvidenceRecord
	}
	candidates := make([]candidate, 0)
	for rows.Next() {
		var item candidate
		var encoded string
		if err := rows.Scan(&item.key, &item.version, &encoded); err != nil {
			rows.Close()
			return ctx
		}
		if err := decodeJSON([]byte(encoded), &item.record); err != nil {
			rows.Close()
			return ctx
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return ctx
	}
	if err := rows.Close(); err != nil {
		return ctx
	}
	checks := applyEvidenceChecks{service: s, projectID: projectID, byKey: make(map[string]applyEvidenceCheck, len(candidates))}
	// The same content-addressed payload may have several semantic Evidence keys.
	verified := make(map[EvidenceRecord]bool)
	for _, item := range candidates {
		valid, ok := verified[item.record]
		if !ok {
			var err error
			valid, err = s.evidenceIntegrityValid(projectID, item.record)
			if err != nil {
				continue
			}
			verified[item.record] = valid
		}
		checks.byKey[item.key] = applyEvidenceCheck{version: item.version, record: item.record, valid: valid}
	}
	return context.WithValue(ctx, applyEvidenceContextKey{}, checks)
}

func (s *Service) applyEvidenceIntegrityValid(ctx context.Context, projectID, key string, version int, record EvidenceRecord) (bool, error) {
	if checks, ok := ctx.Value(applyEvidenceContextKey{}).(applyEvidenceChecks); ok && checks.service == s && checks.projectID == projectID {
		if prior, found := checks.byKey[key]; found && prior.version == version && prior.record == record {
			return prior.valid, nil
		}
	}
	return s.evidenceIntegrityValid(projectID, record)
}
