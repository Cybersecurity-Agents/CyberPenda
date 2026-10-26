package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pentest/internal/owner"
	"pentest/internal/session"
	"pentest/internal/steering"
)

func TestSessionListBatchKeepsSelectionSteeringAndPermissionOwners(t *testing.T) {
	server, err := NewServer(Config{DBPath: filepath.Join(t.TempDir(), "pentest.db"), RuntimeRoot: t.TempDir(), DisableBuiltinSkills: true})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ids := make([]string, 0, 2)
	for _, model := range []string{"first-model", "second-model"} {
		created, err := server.sessions.Create(session.CreateRequest{Input: model})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, created.ID)
		config := map[string]any{"snapshot_version": 1, "runtime_plugin_id": "codex", "runtime_turn_selection": map[string]any{"model": model, "model_provider_id": "provider"}, "settings": map[string]any{}}
		if _, err := server.sessions.CreateContinuation(created.ID, "profile", "codex", session.RunnerSandbox, config); err != nil {
			t.Fatal(err)
		}
		if _, err := server.steering.Accept(context.Background(), owner.KindSession, created.ID, steering.AcceptRequest{RequestID: model, Message: "steer", Mode: owner.SteeringModeInTurnSteer}, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := server.sessions.AppendEvent(created.ID, session.EventKindPermission, session.EventPayload{"phase": "provider_permission_requested", "permission_request_id": "shared-permission"}); err != nil {
			t.Fatal(err)
		}
	}
	// Applying the same provider permission ID in one Session must not clear it
	// in the other Session.
	if _, err := server.sessions.AppendEvent(ids[0], session.EventKindLifecycle, session.EventPayload{"phase": "provider_permission_response_applied", "permission_request_id": "shared-permission"}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/sessions", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list = %d %s", response.Code, response.Body.String())
	}
	var body struct {
		Sessions []session.Session `json:"sessions"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Sessions) != 2 {
		t.Fatalf("Sessions = %d", len(body.Sessions))
	}
	for _, found := range body.Sessions {
		wantModel := "second-model"
		wantPermissions := 1
		if found.ID == ids[0] {
			wantModel = "first-model"
			wantPermissions = 0
		}
		if found.RuntimeConfiguration == nil || found.RuntimeConfiguration.Model != wantModel || found.RuntimeControls.TurnSelection == nil || found.RuntimeControls.TurnSelection.Model != wantModel {
			t.Fatalf("selection = %#v", found)
		}
		if found.RuntimeControls.NativeSteerRequestID != wantModel || found.RuntimeControls.NativeSteerState != "pending" {
			t.Fatalf("Steering = %#v", found.RuntimeControls)
		}
		if len(found.RuntimeControls.ProviderPermissions) != wantPermissions {
			t.Fatalf("permission count = %d for %s", len(found.RuntimeControls.ProviderPermissions), found.ID)
		}
	}
}
