package session

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DecorationData contains persisted inputs for one Session list projection.
// Runtime health remains owned by the live Runtime Harness.
type DecorationData struct {
	Active           *Continuation
	Latest           *Continuation
	Config           map[string]any
	PermissionEvents []Event
}

// DecorationDataForOwners reads bounded latest metadata for all requested
// Sessions. Ordinary Runtime Output payloads are excluded from permission reads.
func (s *Service) DecorationDataForOwners(sessionIDs []string) (map[string]DecorationData, error) {
	result := make(map[string]DecorationData, len(sessionIDs))
	ids := make([]string, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if _, ok := result[id]; !ok {
			result[id] = DecorationData{}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return result, nil
	}
	// Each statement releases the Store's single connection. Do not wrap this
	// read projection in a transaction: Store Begin uses an immediate write lock.
	for offset := 0; offset < len(ids); offset += 500 {
		end := min(offset+500, len(ids))
		args := make([]any, 0, end-offset)
		for _, id := range ids[offset:end] {
			args = append(args, id)
		}
		placeholders := strings.TrimSuffix(strings.Repeat("(?),", len(args)), ",")
		requested := `WITH requested(id) AS (VALUES ` + placeholders + `) `
		rows, err := s.db.Query(requested+`SELECT id,session_id,number,runtime_profile_id,runtime_provider,runner,status,container_id,native_session_id,native_session_path,runtime_config_version_id,started_at,updated_at,ended_at
   FROM session_continuations WHERE id IN (
    SELECT (SELECT id FROM session_continuations WHERE session_id=requested.id ORDER BY number DESC LIMIT 1) FROM requested
    UNION
    SELECT (SELECT id FROM session_continuations WHERE session_id=requested.id AND status IN ('pending','running') ORDER BY number DESC LIMIT 1) FROM requested
   )`, args...)
		if err != nil {
			return nil, fmt.Errorf("read Session list Continuations: %w", err)
		}
		for rows.Next() {
			continuation, err := scanContinuation(rows)
			if err != nil {
				rows.Close()
				return nil, err
			}
			data := result[continuation.SessionID]
			if data.Latest == nil || continuation.Number > data.Latest.Number {
				value := continuation
				data.Latest = &value
			}
			if continuation.Status == RuntimeStatusPending || continuation.Status == RuntimeStatusRunning {
				value := continuation
				data.Active = &value
			}
			result[continuation.SessionID] = data
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		rows, err = s.db.Query(requested+`SELECT session_id,config_json FROM session_runtime_config_versions WHERE id IN (
   SELECT (SELECT id FROM session_runtime_config_versions WHERE session_id=requested.id ORDER BY version DESC LIMIT 1) FROM requested
  )`, args...)
		if err != nil {
			return nil, fmt.Errorf("read latest Session list configuration: %w", err)
		}
		for rows.Next() {
			var id, raw string
			if err := rows.Scan(&id, &raw); err != nil {
				rows.Close()
				return nil, err
			}
			var config map[string]any
			if json.Unmarshal([]byte(raw), &config) == nil {
				data := result[id]
				data.Config = config
				result[id] = data
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		rows, err = s.db.Query(`SELECT id,session_id,seq,kind,payload_json,created_at FROM session_events
   WHERE session_id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")+`)
   AND kind IN ('permission','lifecycle')
   AND json_extract(payload_json,'$.permission_request_id') IS NOT NULL
   AND json_extract(payload_json,'$.phase') IN ('provider_permission_requested','provider_permission_response_applied')
   ORDER BY session_id,seq ASC`, args...)
		if err != nil {
			return nil, fmt.Errorf("read Session list permissions: %w", err)
		}
		events, err := scanEvents(rows)
		rows.Close()
		if err != nil {
			return nil, err
		}
		for _, event := range events {
			data := result[event.SessionID]
			data.PermissionEvents = append(data.PermissionEvents, event)
			result[event.SessionID] = data
		}
	}
	return result, nil
}
