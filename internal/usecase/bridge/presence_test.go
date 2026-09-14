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
