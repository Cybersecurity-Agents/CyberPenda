package session

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AppendRuntimeOutputs stores one bounded Runtime output batch atomically.
// It validates the owner and Continuations once, allocates the sequence range
// once, and advances Session activity once for all output rows.
func (s *Service) AppendRuntimeOutputs(id string, payloads []EventPayload) ([]Event, error) {
	events := make([]Event, 0, len(payloads))
	if len(payloads) == 0 {
		return events, nil
	}
	found, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if found.Lifecycle != LifecycleOpen {
		return nil, ErrSessionNotOpen
	}
	checked := map[string]bool{}
	for _, payload := range payloads {
		continuationID, _ := payload["continuation_id"].(string)
		if strings.TrimSpace(continuationID) == "" || checked[continuationID] {
			continue
		}
		c, err := s.Continuation(continuationID)
		if err != nil || c.SessionID != id {
			return nil, ErrContinuationNotFound
		}
		checked[continuationID] = true
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin Session output batch: %w", err)
	}
	defer tx.Rollback()
	var max sql.NullInt64
	if err := tx.QueryRow(`SELECT MAX(seq) FROM session_events WHERE session_id=?`, id).Scan(&max); err != nil {
		return nil, fmt.Errorf("read Session Event sequence: %w", err)
	}
	now := time.Now().UTC()
	for index, payload := range payloads {
		event, err := appendEventAtSequenceTx(tx, id, EventKindRuntimeOutput, payload, now, int(max.Int64)+index+1)
		if err != nil {
			return nil, fmt.Errorf("store Session output batch: %w", err)
		}
		events = append(events, event)
	}
	if err := touchSessionActivityTx(tx, id, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit Session output batch: %w", err)
	}
	return events, nil
}
