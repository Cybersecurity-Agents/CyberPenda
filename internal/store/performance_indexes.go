package store

import (
	"database/sql"
	"fmt"
)

// ensurePerformanceIndexes adds optional query indexes after baseline validation.
// These indexes do not change the Store data model or baseline checksum.
func ensurePerformanceIndexes(db *sql.DB) error {
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS idx_session_events_permissions ON session_events(session_id, seq ASC) WHERE kind IN ('permission','lifecycle')`,
		`CREATE INDEX IF NOT EXISTS idx_session_events_session_kind_seq ON session_events(session_id, kind, seq ASC)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("prepare Store performance index: %w", err)
		}
	}
	return nil
}
