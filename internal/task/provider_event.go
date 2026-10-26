package task

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AppendProviderEvent binds an unsolicited Runtime Event to the active
// Continuation at persistence time. Resolve the owner and binding in the same
// transaction as the append, without decoding the full Task twice per event.
func (s *Service) AppendProviderEvent(taskID string, kind EventKind, payload EventPayload) (Event, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Event{}, fmt.Errorf("begin provider event: %w", err)
	}
	defer tx.Rollback()
	var continuationID string
	err = tx.QueryRow(`SELECT COALESCE((SELECT id FROM task_continuations WHERE task_id=tasks.id AND status IN ('pending','running','paused') ORDER BY number DESC LIMIT 1),'') FROM tasks WHERE id=? AND deleted_at=''`, taskID).Scan(&continuationID)
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, fmt.Errorf("resolve provider event owner: %w", err)
	}
	event, err := appendTaskContinuationEventTx(tx, taskID, continuationID, kind, payload, time.Now().UTC())
	if err != nil {
		return Event{}, err
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("commit provider event: %w", err)
	}
	return event, nil
}
