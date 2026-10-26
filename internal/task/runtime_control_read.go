package task

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// RuntimeControlEvents reads permission history and the latest recovery boundary.
// Runtime output and earlier recovery boundaries are not part of this projection.
func (s *Service) RuntimeControlEvents(taskID string) ([]Event, error) {
	rows, err := s.db.Query(`SELECT id, task_id, continuation_id, attempt_node_id, seq, kind, payload_json, created_at
 FROM task_events WHERE task_id=? AND kind='lifecycle' AND (
 json_extract(payload_json,'$.phase') IN ('provider_permission_requested','provider_permission_response_applied')
 OR seq=(SELECT seq FROM task_events WHERE task_id=? AND kind='lifecycle'
 AND json_extract(payload_json,'$.phase') IN ('started','provider_session_recovery_required') ORDER BY seq DESC LIMIT 1)) ORDER BY seq ASC`, taskID, taskID)
	if err != nil {
		return nil, fmt.Errorf("read Runtime controls: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// LatestRuntimeConfig reads only the last captured Runtime configuration.
func (s *Service) LatestRuntimeConfig(taskID string) (*RuntimeConfigVersion, error) {
	var version RuntimeConfigVersion
	var configJSON, createdAt string
	err := s.db.QueryRow(`SELECT id,task_id,version,runtime_profile_id,config_json,created_at FROM task_runtime_config_versions WHERE task_id=? ORDER BY version DESC LIMIT 1`, taskID).Scan(&version.ID, &version.TaskID, &version.Version, &version.RuntimeProfileID, &configJSON, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read latest Runtime configuration: %w", err)
	}
	if err := json.Unmarshal([]byte(configJSON), &version.Config); err != nil {
		return nil, fmt.Errorf("decode latest Runtime configuration: %w", err)
	}
	version.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse latest Runtime configuration time: %w", err)
	}
	return &version, nil
}
