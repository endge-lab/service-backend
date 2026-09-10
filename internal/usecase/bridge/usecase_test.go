package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type accessStub struct {
	ports.BridgeAccessRepository
	revoked bool
}

func (s *accessStub) BridgeUser(_ context.Context, id, _ string) (entities.Actor, error) {
	if s.revoked {
		return entities.Actor{}, fmt.Errorf("revoked")
	}
	return entities.Actor{ID: id, DisplayName: "Developer"}, nil
}

type workspaceStub struct {
	ports.WorkspaceRepository
	role string
}

func (s *workspaceStub) GetWorkspace(_ context.Context, id string) (*entities.Workspace, error) {
	return &entities.Workspace{ID: id, Identity: id, Active: true}, nil
}
func (s *workspaceStub) WorkspaceRole(context.Context, string, string, bool) (string, error) {
	return s.role, nil
}

type grantsStub struct{ ports.AccessControlRepository }

func (*grantsStub) IsPlatformAdmin(context.Context, string) (bool, error) { return false, nil }

type deliveryStub struct {
	messages map[string][]entities.BridgeMessage
}

func (s *deliveryStub) Send(id string, m entities.BridgeMessage) bool {
	s.messages[id] = append(s.messages[id], m)
	return true
}
func (*deliveryStub) Close(string) {}

func fixture(t *testing.T) (*UseCase, *accessStub, *workspaceStub, *deliveryStub) {
	t.Helper()
	access := &accessStub{}
	workspaces := &workspaceStub{role: "editor"}
	delivery := &deliveryStub{messages: map[string][]entities.BridgeMessage{}}
	u := NewUseCase(delivery, access, workspaces, &grantsStub{}, true)
	for _, p := range []struct{ id, role string }{{"config", "configurator"}, {"client", "client"}, {"other", "client"}} {
		err := u.Join(context.Background(), p.id, p.role, entities.BridgePrincipal{UserID: "user", ExpiresAt: time.Now().Add(time.Hour)}, entities.BridgeHello{Protocol: 1, WorkspaceIdentity: "workspace", Debug: true})
		if err != nil {
			t.Fatal(err)
		}
	}
	return u, access, workspaces, delivery
}

func approve(t *testing.T, u *UseCase) string {
	t.Helper()
	if err := u.Handle(context.Background(), "config", entities.BridgeMessage{Type: "requestSession", ID: "request", TargetID: "client"}); err != nil {
		t.Fatal(err)
	}
	var id string
	for key := range u.sessions {
		id = key
	}
	if err := u.Handle(context.Background(), "client", entities.BridgeMessage{Type: "acceptSession", SessionID: id, Accepted: true}); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestConsentMembershipAndRevocation(t *testing.T) {
	u, access, _, delivery := fixture(t)
	if err := u.Handle(context.Background(), "config", entities.BridgeMessage{Type: "getSnapshot", ID: "before", SessionID: "missing"}); err == nil {
		t.Fatal("snapshot before consent")
	}
	id := approve(t, u)
	if err := u.Handle(context.Background(), "config", entities.BridgeMessage{Type: "getSnapshot", ID: "snapshot", SessionID: id}); err != nil {
		t.Fatal(err)
	}
	var commandID string
	for key := range u.commands {
		commandID = key
	}
	payload := json.RawMessage(`{"telemetry":[]}`)
	if err := u.Handle(context.Background(), "other", entities.BridgeMessage{Type: "commandResult", ID: commandID, SessionID: id, Data: payload}); err == nil {
		t.Fatal("unrelated peer supplied response")
	}
	access.revoked = true
	_ = u.Handle(context.Background(), "client", entities.BridgeMessage{Type: "commandResult", ID: commandID, SessionID: id, Data: payload})
	for _, m := range delivery.messages["config"] {
		if m.ID == "snapshot" && m.Error == "" {
			t.Fatal("snapshot leaked after revocation")
		}
	}
	if len(u.sessions) != 0 || len(u.commands) != 0 {
		t.Fatal("revocation retained session resources")
	}
}

func TestViewerAndCrossWorkspaceDenied(t *testing.T) {
	u, _, workspaces, _ := fixture(t)
	workspaces.role = "viewer"
	if err := u.Handle(context.Background(), "config", entities.BridgeMessage{Type: "requestSession", ID: "request", TargetID: "client"}); err == nil {
		t.Fatal("viewer can debug")
	}
	workspaces.role = "editor"
	u.peers["client"].Workspace = "another"
	if err := u.Handle(context.Background(), "config", entities.BridgeMessage{Type: "requestSession", ID: "request", TargetID: "client"}); err == nil {
		t.Fatal("cross workspace debug")
	}
}

func TestExpiryAndDisconnectReleaseAllResources(t *testing.T) {
	u, _, _, _ := fixture(t)
	id := approve(t, u)
	if err := u.Handle(context.Background(), "config", entities.BridgeMessage{Type: "getSnapshot", ID: "snapshot", SessionID: id}); err != nil {
		t.Fatal(err)
	}
	u.sessions[id].Deadline = time.Now().Add(-time.Second)
	u.Sweep()
	if len(u.sessions) != 0 || len(u.commands) != 0 {
		t.Fatal("expired session retained requests")
	}
	u.peers["config"].LastRequest = time.Time{}
	id = approve(t, u)
	if err := u.Handle(context.Background(), "config", entities.BridgeMessage{Type: "getSnapshot", ID: "snapshot2", SessionID: id}); err != nil {
		t.Fatal(err)
	}
	u.Leave("client")
	u.Leave("client")
	if len(u.sessions) != 0 || len(u.commands) != 0 {
		t.Fatal("disconnect retained resources")
	}
}

func TestClientReceivesNoRosterAndDebugDefaultDenied(t *testing.T) {
	_, _, _, delivery := fixture(t)
	for _, m := range delivery.messages["client"] {
		if m.Type == "configurators" || m.Type == "clients" {
			t.Fatal("unauthenticated roster")
		}
	}
	u := NewUseCase(delivery, &accessStub{}, &workspaceStub{role: "editor"}, &grantsStub{}, false)
	if err := u.Join(context.Background(), "client", "client", entities.BridgePrincipal{}, entities.BridgeHello{Protocol: 1, WorkspaceIdentity: "workspace", Debug: true}); err == nil {
		t.Fatal("debug disabled accepted client")
	}
}

func TestContextSyncRoutesOpaquePayloadsOnlyWithinApprovedSession(t *testing.T) {
	u, _, _, delivery := fixture(t)
	id := approve(t, u)
	payload := json.RawMessage(`{"type":"context:set-locale","payload":{"locale":"en"}}`)
	for _, kind := range []string{"startContextSync", "executeCommand", "refreshInspection", "setInspectionOptions"} {
		if err := u.Handle(context.Background(), "config", entities.BridgeMessage{Type: kind, ID: kind, SessionID: id, Data: payload}); err != nil {
			t.Fatal(err)
		}
		message := delivery.messages["client"][len(delivery.messages["client"])-1]
		if message.Type != kind || message.ID == kind || message.SessionID != id {
			t.Fatalf("invalid correlated request: %+v", message)
		}
		if (kind == "executeCommand" || kind == "setInspectionOptions") && string(message.Data) != string(payload) {
			t.Fatal("command payload changed in transport")
		}
	}
	event := json.RawMessage(`{"sequence":1,"event":{"name":"custom:event","payload":{"opaque":true}}}`)
	message := entities.BridgeMessage{Type: "clientEvent", SessionID: id, Data: event, TargetID: "other", Error: "untrusted"}
	for _, sender := range []string{"config", "other"} {
		if err := u.Handle(context.Background(), sender, message); err == nil {
			t.Fatalf("event accepted from %s", sender)
		}
	}
	if err := u.Handle(context.Background(), "client", message); err != nil {
		t.Fatal(err)
	}
	forwarded := delivery.messages["config"][len(delivery.messages["config"])-1]
	if forwarded.Type != "clientEvent" || string(forwarded.Data) != string(event) || forwarded.TargetID != "" || forwarded.Error != "" {
		t.Fatalf("invalid forwarded event: %+v", forwarded)
	}
	u.Leave("config")
	if err := u.Handle(context.Background(), "client", message); err == nil {
		t.Fatal("event accepted after session ended")
	}
}

func TestContextSyncChecksPayloadBoundsAndRevocation(t *testing.T) {
	for _, kind := range []string{"clientEvent", "executeCommand", "inspectionSnapshot", "setInspectionOptions"} {
		t.Run(kind, func(t *testing.T) {
			u, _, workspaces, delivery := fixture(t)
			id := approve(t, u)
			sender, receiver, limit := "client", "config", maxEventBytes
			if kind == "executeCommand" || kind == "setInspectionOptions" {
				sender, receiver, limit = "config", "client", maxCommandBytes
			}
			if kind == "inspectionSnapshot" {
				limit = maxInspectionBytes
			}
			for _, payload := range []json.RawMessage{nil, json.RawMessage(`{`), json.RawMessage(`"` + strings.Repeat("x", limit) + `"`)} {
				before := len(delivery.messages[receiver])
				if err := u.Handle(context.Background(), sender, entities.BridgeMessage{Type: kind, ID: "request", SessionID: id, Data: payload}); err == nil {
					t.Fatal("invalid payload accepted")
				}
				if len(delivery.messages[receiver]) != before {
					t.Fatal("invalid payload forwarded")
				}
			}
			workspaces.role = "viewer"
			if err := u.Handle(context.Background(), sender, entities.BridgeMessage{Type: kind, ID: "revoked", SessionID: id, Data: json.RawMessage(`{}`)}); err == nil {
				t.Fatal("revoked controller retained access")
			}
			if len(u.sessions) != 0 || len(u.commands) != 0 {
				t.Fatal("revocation retained resources")
			}
		})
	}
}

// Large snapshots retain the same consent, sender and session boundary as small events.
func TestInspectionSnapshotRouting(t *testing.T) {
	u, _, _, delivery := fixture(t)
	id := approve(t, u)
	payload := json.RawMessage(`{"sequence":1,"update":{"kind":"data","data":"` + strings.Repeat("x", maxEventBytes+1) + `","generatedAt":1}}`)
	m := entities.BridgeMessage{Type: "inspectionSnapshot", SessionID: id, Data: payload, TargetID: "other", Error: "untrusted"}
	for _, sender := range []string{"config", "other"} {
		if err := u.Handle(context.Background(), sender, m); err == nil {
			t.Fatalf("snapshot accepted from %s", sender)
		}
	}
	if err := u.Handle(context.Background(), "client", m); err != nil {
		t.Fatal(err)
	}
	forwarded := delivery.messages["config"][len(delivery.messages["config"])-1]
	if forwarded.Type != m.Type || string(forwarded.Data) != string(payload) || forwarded.TargetID != "" || forwarded.Error != "" {
		t.Fatal("snapshot routing changed payload or leaked untrusted envelope fields")
	}
	m.Type = "clientEvent"
	if err := u.Handle(context.Background(), "client", m); err == nil {
		t.Fatal("snapshot allowance widened ordinary event limit")
	}
	u.Leave("config")
	m.Type = "inspectionSnapshot"
	if err := u.Handle(context.Background(), "client", m); err == nil {
		t.Fatal("snapshot accepted after disconnect")
	}
}
