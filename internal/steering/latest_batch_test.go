package steering

import (
	"context"
	"pentest/internal/owner"
	"testing"
)

func TestLatestForOwnersKeepsOwnerKindAndLatestQueueOrder(t *testing.T) {
	s := newTestService(t)
	for _, row := range []struct {
		kind        owner.Kind
		id, request string
	}{{owner.KindSession, "one", "first"}, {owner.KindSession, "one", "latest"}, {owner.KindTask, "one", "other-kind"}, {owner.KindSession, "two", "other-owner"}} {
		if _, err := s.Accept(context.Background(), row.kind, row.id, AcceptRequest{RequestID: row.request, Message: "message", Mode: owner.SteeringModeInTurnSteer}, nil); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.LatestForOwners(owner.KindSession, []string{"one", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["one"].RequestID != "latest" {
		t.Fatalf("latest = %#v", got)
	}
}
