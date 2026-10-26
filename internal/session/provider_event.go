package session

import (
	"database/sql"
	"errors"
	"fmt"
)

// AppendProviderEvent resolves the active Runtime Continuation and appends the
// Event in one transaction. Provider payloads cannot select another binding.
func (s *Service) AppendProviderEvent(sessionID string, kind EventKind, payload EventPayload) (Event, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Event{}, fmt.Errorf("begin Session provider event: %w", err)
	}
	defer tx.Rollback()
	var lifecycle, continuationID string
	err = tx.QueryRow(`SELECT lifecycle,COALESCE((SELECT id FROM session_continuations WHERE session_id=sessions.id AND status IN ('pending','running') ORDER BY number DESC LIMIT 1),'') FROM sessions WHERE id=?`, sessionID).Scan(&lifecycle, &continuationID)
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err != nil {
		return Event{}, fmt.Errorf("resolve Session provider event owner: %w", err)
	}
	if Lifecycle(lifecycle) != LifecycleOpen {
		return Event{}, ErrSessionNotOpen
	}
	filtered := make(EventPayload, len(payload)+1)
	for key, value := range payload {
		if key != "continuation_id" {
			filtered[key] = value
		}
	}
	if continuationID != "" {
		filtered["continuation_id"] = continuationID
	}
	event, err := s.AppendEventTx(tx, sessionID, kind, filtered)
	if err != nil {
		return Event{}, err
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("commit Session provider event: %w", err)
	}
	return event, nil
}
