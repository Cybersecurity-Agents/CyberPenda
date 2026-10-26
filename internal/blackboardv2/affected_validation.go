package blackboardv2

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// collectAffectedConclusions includes explicit keys and Findings supported by a
// changed source. Call before and after mutation so removed support is retained
// in the validation set. Finding support is defined by direct Fact/Evidence
// edges; dependent Fact basis validation remains a separate unchanged check.
func collectAffectedConclusions(ctx context.Context, tx *sql.Tx, projectID string, keys []string, affected map[string]bool) error {
	unique := make(map[string]bool)
	for _, key := range keys {
		if key != "" {
			unique[key] = true
			affected[key] = true
		}
	}
	if len(unique) == 0 {
		return nil
	}
	sources := make([]string, 0, len(unique))
	for key := range unique {
		sources = append(sources, key)
	}
	raw, err := json.Marshal(sources)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT rel.to_key FROM blackboard_v2_relationships AS rel
 JOIN blackboard_v2_records AS target ON target.project_id=rel.project_id AND target.key=rel.to_key AND target.type='finding'
 WHERE rel.project_id=? AND rel.from_key IN (SELECT value FROM json_each(?)) AND rel.relation IN ('supports','evidences')`, projectID, string(raw))
	if err != nil {
		return fmt.Errorf("read affected Finding support: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return err
		}
		affected[key] = true
	}
	return rows.Err()
}
