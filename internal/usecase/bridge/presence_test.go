package bridge

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestPresenceAcrossWorkspacesKeepsDebugScoped(t *testing.T) {
	u, _, workspaces, delivery := fixture(t)
	workspaces.names = map[string]string{"workspace": "First workspace", "second": "Second workspace"}
	if !u.Check(context.Background(), "config") {
		t.Fatal("existing connection failed authorization")
	}
	if err := u.Join(context.Background(), "second-config", "configurator",
		entities.BridgePrincipal{UserID: "second-user", ExpiresAt: time.Now().Add(time.Hour)},
		entities.BridgeHello{Protocol: 1, WorkspaceIdentity: "second", Debug: true}); err != nil {
		t.Fatal(err)
	}
	lastMessage := func(id, kind string) entities.BridgeMessage {
		t.Helper()
		messages := delivery.messages[id]
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Type == kind {
				return messages[i]
			}
		}
		t.Fatalf("missing %s for %s", kind, id)
		return entities.BridgeMessage{}
	}
	for _, id := range []string{"config", "second-config"} {
		var roster []entities.BridgeConfigurator
		if err := json.Unmarshal(lastMessage(id, "configurators").Data, &roster); err != nil {
			t.Fatal(err)
		}
		if len(roster) != 2 || roster[0].WorkspaceDisplayName != "First workspace" || roster[1].WorkspaceDisplayName != "Second workspace" {
			t.Fatalf("incomplete backend roster for %s: %+v", id, roster)
		}
	}
	var clients []entities.BridgeClient
	if err := json.Unmarshal(lastMessage("second-config", "clients").Data, &clients); err != nil || len(clients) != 0 {
		t.Fatalf("debug clients leaked across workspaces: %+v, %v", clients, err)
	}
	if err := u.Handle(context.Background(), "second-config", entities.BridgeMessage{Type: "requestSession", ID: "cross-workspace", TargetID: "client"}); err == nil {
		t.Fatal("cross-workspace debug was allowed")
	}
	workspaces.names["second"] = "Renamed workspace"
	if !u.Check(context.Background(), "second-config") {
		t.Fatal("renamed workspace failed authorization")
	}
	var roster []entities.BridgeConfigurator
	if err := json.Unmarshal(lastMessage("config", "configurators").Data, &roster); err != nil || len(roster) != 2 || roster[1].WorkspaceDisplayName != "Renamed workspace" {
		t.Fatalf("workspace rename did not reach another workspace: %+v, %v", roster, err)
	}
	u.Leave("second-config")
	if err := json.Unmarshal(lastMessage("config", "configurators").Data, &roster); err != nil || len(roster) != 1 {
		t.Fatalf("disconnect did not reach another workspace: %+v, %v", roster, err)
	}
	for _, m := range delivery.messages["client"] {
		if m.Type == "configurators" {
			t.Fatal("presence exposed to unauthenticated debug client")
		}
	}
}

// Independent discovery uses target permissions before consent and on every subsequent message.
func TestIndependentDebuggerTargetsAuthorizedWorkspaces(t *testing.T) {
	ctx := context.Background()
	u, _, workspaces, delivery := fixture(t)
	workspaces.names = map[string]string{"workspace": "AODB", "second": "Second", "forbidden": "Private"}
	workspaces.roles = map[string]string{"user:workspace": "editor", "inspector:workspace": "editor", "inspector:second": "viewer"}
	if err := u.Join(ctx, "second-client", "client", entities.BridgePrincipal{}, entities.BridgeHello{Protocol: 1, WorkspaceIdentity: "second", Debug: true}); err != nil {
		t.Fatal(err)
	}
	if err := u.Join(ctx, "private-client", "client", entities.BridgePrincipal{}, entities.BridgeHello{Protocol: 1, WorkspaceIdentity: "forbidden", Debug: true}); err != nil {
		t.Fatal(err)
	}
	if err := u.Join(ctx, "independent", "configurator", entities.BridgePrincipal{UserID: "inspector", ExpiresAt: time.Now().Add(time.Hour)}, entities.BridgeHello{Protocol: 1, Debug: true, AllWorkspaces: true}); err != nil {
		t.Fatal(err)
	}
	roster := func() []entities.BridgeClient {
		t.Helper()
		messages := delivery.messages["independent"]
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Type == "clients" {
				var result []entities.BridgeClient
				if err := json.Unmarshal(messages[i].Data, &result); err != nil {
					t.Fatal(err)
				}
				return result
			}
		}
		t.Fatal("no roster")
		return nil
	}
	if clients := roster(); len(clients) != 2 || clients[0].WorkspaceDisplayName != "AODB" {
		t.Fatalf("unexpected authorized clients: %+v", clients)
	}
	for _, target := range []string{"second-client", "private-client"} {
		if err := u.Handle(ctx, "independent", entities.BridgeMessage{Type: "requestSession", ID: "denied", TargetID: target}); err == nil {
			t.Fatal("unauthorized target accepted")
		}
	}
	if err := u.Handle(ctx, "independent", entities.BridgeMessage{Type: "requestSession", ID: "join", TargetID: "client"}); err != nil {
		t.Fatal(err)
	}
	var sessionID string
	for id, session := range u.sessions {
		if session.ConfiguratorID == "independent" {
			sessionID = id
		}
	}
	if err := u.Handle(ctx, "independent", entities.BridgeMessage{Type: "startContextSync", ID: "before", SessionID: sessionID}); err == nil {
		t.Fatal("snapshot before consent")
	}
	if err := u.Handle(ctx, "client", entities.BridgeMessage{Type: "acceptSession", SessionID: sessionID, Accepted: true}); err != nil {
		t.Fatal(err)
	}
	if err := u.Handle(ctx, "independent", entities.BridgeMessage{Type: "startContextSync", ID: "snapshot", SessionID: sessionID}); err != nil {
		t.Fatal(err)
	}
	var commandID string
	for id := range u.commands {
		commandID = id
	}
	payload := json.RawMessage(`{"format":"endge-bundle"}`)
	if err := u.Handle(ctx, "client", entities.BridgeMessage{Type: "commandResult", ID: commandID, SessionID: sessionID, Data: payload}); err != nil {
		t.Fatal(err)
	}
	workspaces.roles["inspector:workspace"] = "viewer"
	before := len(delivery.messages["independent"])
	if err := u.Handle(ctx, "client", entities.BridgeMessage{Type: "inspectionChunk", SessionID: sessionID, Data: payload}); err == nil {
		t.Fatal("chunk forwarded after target permission revocation")
	}
	for _, message := range delivery.messages["independent"][before:] {
		if message.Type == "inspectionChunk" {
			t.Fatal("revoked data leaked")
		}
	}
	if len(u.sessions) != 0 || len(u.commands) != 0 || len(roster()) != 0 {
		t.Fatal("revocation retained resources or visible clients")
	}
	workspaces.roles["inspector:second"] = "editor"
	if !u.Check(ctx, "independent") {
		t.Fatal("independent session requires an authoring workspace")
	}
	if clients := roster(); len(clients) != 1 || clients[0].InstanceID != "second-client" {
		t.Fatalf("new target grant missing: %+v", clients)
	}
}

func TestIndependentDiscoveryCannotBeRequestedByClient(t *testing.T) {
	u, _, _, _ := fixture(t)
	if err := u.Join(context.Background(), "bad", "client", entities.BridgePrincipal{}, entities.BridgeHello{Protocol: 1, Debug: true, AllWorkspaces: true}); err == nil {
		t.Fatal("client escalated discovery scope")
	}
}
