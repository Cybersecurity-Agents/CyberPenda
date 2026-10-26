package daemon

import (
	"pentest/internal/owner"
	"pentest/internal/session"
)

// decorateSessions separates persisted batch inputs from live Runtime health.
// Every owner uses the same pure projection as the Session detail endpoint.
func (server *Server) decorateSessions(found []session.Session) ([]session.Session, error) {
	ids := make([]string, 0, len(found))
	for _, item := range found {
		ids = append(ids, item.ID)
	}
	data, err := server.sessions.DecorationDataForOwners(ids)
	if err != nil {
		return nil, err
	}
	steeringRecords, _ := server.steering.LatestForOwners(owner.KindSession, ids)
	decorated := make([]session.Session, 0, len(found))
	for _, item := range found {
		control := steeringControlProjection{}
		if record := steeringRecords[item.ID]; record != nil {
			control = steeringControlProjection{requestID: record.RequestID, state: steeringOutcomeFromRecord(record), errorCode: string(record.ErrorCode), error: record.ErrorMessage, nonTerminal: !record.State.Terminal()}
		}
		projected, err := server.decorateSessionFromData(item, data[item.ID], control)
		if err != nil {
			return nil, err
		}
		decorated = append(decorated, projected)
	}
	return decorated, nil
}
