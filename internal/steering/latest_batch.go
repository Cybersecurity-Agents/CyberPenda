package steering

import (
	"fmt"
	"pentest/internal/owner"
	"strings"
)

// LatestForOwners loads the latest Accepted Steering record for each requested
// Runtime Owner. It never loads records from another owner kind.
func (s *Service) LatestForOwners(kind owner.Kind, ownerIDs []string) (map[string]*owner.SteeringRecord, error) {
	result := make(map[string]*owner.SteeringRecord)
	for offset := 0; offset < len(ownerIDs); offset += 500 {
		end := min(offset+500, len(ownerIDs))
		args := make([]any, 0, end-offset+1)
		for _, id := range ownerIDs[offset:end] {
			args = append(args, id)
		}
		args = append(args, string(kind))
		rows, err := s.db.Query(`WITH requested(id) AS (VALUES `+strings.TrimSuffix(strings.Repeat("(?),", end-offset), ",")+`)
   SELECT `+steeringRecordColumns+` FROM accepted_steering WHERE id IN (
    SELECT (SELECT id FROM accepted_steering WHERE owner_kind=? AND owner_id=requested.id ORDER BY queue_order DESC LIMIT 1) FROM requested
   )`, args...)
		if err != nil {
			return nil, fmt.Errorf("read latest owner Steering: %w", err)
		}
		for rows.Next() {
			record, err := scanSteeringRecordRows(rows)
			if err != nil {
				rows.Close()
				return nil, err
			}
			result[record.OwnerID] = record
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
