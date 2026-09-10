// Package bridge owns ephemeral participants and explicitly approved debug sessions.
package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

const requestTTL = 45 * time.Second
const sessionTTL = 30 * time.Minute
const maxEventBytes = 1024 * 1024
const maxInspectionBytes = 15 * 1024 * 1024
const maxCommandBytes = 64 * 1024

var hashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type participant struct {
	ID, Role, Workspace, Label string
	Principal                  entities.BridgePrincipal
	Actor                      entities.Actor
	Debug                      bool
	AccessRole                 string
	LastRequest                time.Time
}

type session struct {
	entities.BridgeSession
	RequestID string
	Active    bool
	Deadline  time.Time
}

type command struct {
	SessionID, RequestID string
	Deadline             time.Time
}

// UseCase is the sole owner of logical participants/sessions. Delivery owns only sockets.
type UseCase struct {
	mu           sync.Mutex
	delivery     ports.BridgeDelivery
	access       ports.BridgeAccessRepository
	workspaces   ports.WorkspaceRepository
	grants       ports.AccessControlRepository
	debugEnabled bool
	peers        map[string]*participant
	sessions     map[string]*session
	commands     map[string]command
}

func NewUseCase(delivery ports.BridgeDelivery, access ports.BridgeAccessRepository, workspaces ports.WorkspaceRepository, grants ports.AccessControlRepository, debugEnabled bool) *UseCase {
	return &UseCase{delivery: delivery, access: access, workspaces: workspaces, grants: grants, debugEnabled: debugEnabled,
		peers: make(map[string]*participant), sessions: make(map[string]*session), commands: make(map[string]command)}
}

// Join accepts a server-generated connection ID and a transport-authenticated principal.
func (u *UseCase) Join(ctx context.Context, id, role string, principal entities.BridgePrincipal, hello entities.BridgeHello) error {
	if hello.Protocol != 1 || len(hello.WorkspaceIdentity) == 0 || len(hello.WorkspaceIdentity) > 160 || len(hello.Label) > 160 || (role != "client" && role != "configurator") {
		return fmt.Errorf("Invalid bridge registration")
	}
	if role == "client" && (!u.debugEnabled || !hello.Debug) {
		return fmt.Errorf("Client debug is disabled")
	}
	p := &participant{ID: id, Role: role, Principal: principal, Workspace: hello.WorkspaceIdentity, Label: hello.Label, Debug: hello.Debug && u.debugEnabled}
	if role == "configurator" {
		if p.Principal.ExpiresAt.IsZero() {
			p.Principal.ExpiresAt = time.Now().Add(sessionTTL)
		}
		actor, accessRole, err := u.authorize(ctx, *p)
		if err != nil {
			return err
		}
		p.Actor, p.AccessRole = actor, accessRole
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if _, exists := u.peers[id]; exists {
		return fmt.Errorf("Already registered")
	}
	count := 0
	for _, other := range u.peers {
		if other.Workspace == p.Workspace && other.Role == p.Role {
			count++
		}
	}
	if count >= 64 {
		return fmt.Errorf("Workspace connection limit reached")
	}
	u.peers[id] = p
	u.send(id, "welcome", map[string]any{"protocol": 1, "instanceId": id})
	u.broadcast(p.Workspace)
	return nil
}

// Check refreshes authorization periodically; heartbeat never extends identity expiry.
func (u *UseCase) Check(ctx context.Context, id string) bool {
	u.mu.Lock()
	p := u.peers[id]
	if p == nil {
		u.mu.Unlock()
		return false
	}
	snapshot := *p
	u.mu.Unlock()
	if snapshot.Role == "client" {
		return true
	}
	actor, role, err := u.authorize(ctx, snapshot)
	if err != nil {
		return false
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.peers[id] != p {
		return false
	}
	changed := p.AccessRole != role || p.Actor != actor
	p.Actor, p.AccessRole = actor, role
	if !shared.CanWrite(role) {
		for sessionID, s := range u.sessions {
			if s.ConfiguratorID == id {
				u.end(sessionID, "Debug access revoked")
			}
		}
	}
	if changed {
		u.broadcast(p.Workspace)
	}
	return true
}

// Handle routes only allowlisted operations and verifies session membership on each message.
func (u *UseCase) Handle(ctx context.Context, id string, message entities.BridgeMessage) error {
	if len(message.ID) > 80 || len(message.SessionID) > 80 || len(message.TargetID) > 80 || len(message.Identity) > 160 || len(message.Error) > 512 {
		return fmt.Errorf("Invalid bridge message")
	}
	if !u.Check(ctx, id) {
		return fmt.Errorf("Authorization expired")
	}
	// A late client response must not bypass revocation of the controlling identity.
	if message.Type == "acceptSession" || message.Type == "commandResult" || message.Type == "clientEvent" || message.Type == "inspectionSnapshot" {
		u.mu.Lock()
		s := u.sessions[message.SessionID]
		controller := ""
		if s != nil {
			controller = s.ConfiguratorID
		}
		u.mu.Unlock()
		if controller != "" && !u.Check(ctx, controller) {
			u.Leave(controller)
			u.delivery.Close(controller)
		}
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	p := u.peers[id]
	if p == nil {
		return fmt.Errorf("Connection is not registered")
	}
	u.expire(time.Now())
	switch message.Type {
	case "requestSession":
		return u.requestSession(p, message)
	case "acceptSession":
		return u.acceptSession(p, message)
	case "endSession":
		s := u.sessions[message.SessionID]
		if s == nil || (s.ClientID != id && s.ConfiguratorID != id) {
			return fmt.Errorf("Session is not available")
		}
		u.end(message.SessionID, "Session ended")
		u.result(id, message.ID, nil, "")
		return nil
	case "getSnapshot", "startContextSync", "executeCommand", "runSimulation", "refreshInspection", "setInspectionOptions":
		return u.requestCommand(p, message)
	case "clientEvent", "inspectionSnapshot":
		return u.forwardClientEvent(p, message)
	case "commandResult":
		s := u.sessions[message.SessionID]
		pending, ok := u.commands[message.ID]
		if s == nil || !s.Active || s.ClientID != id || !ok || pending.SessionID != s.SessionID {
			return fmt.Errorf("Command is not pending")
		}
		delete(u.commands, message.ID)
		u.delivery.Send(s.ConfiguratorID, entities.BridgeMessage{Type: "result", ID: pending.RequestID, Data: message.Data, Error: message.Error})
		return nil
	default:
		return fmt.Errorf("Unsupported bridge operation")
	}
}

func (u *UseCase) requestSession(p *participant, m entities.BridgeMessage) error {
	if !p.Debug || p.Role != "configurator" || !shared.CanWrite(p.AccessRole) || m.ID == "" {
		return fmt.Errorf("Debug access denied")
	}
	target := u.peers[m.TargetID]
	if target == nil || target.Role != "client" || !target.Debug || target.Workspace != p.Workspace {
		return fmt.Errorf("Client is not available")
	}
	if time.Since(p.LastRequest) < 5*time.Second {
		return fmt.Errorf("Please wait before requesting another session")
	}
	for _, s := range u.sessions {
		if s.ClientID == target.ID {
			return fmt.Errorf("Client is busy")
		}
	}
	p.LastRequest = time.Now()
	s := &session{BridgeSession: entities.BridgeSession{SessionID: uuid.NewString(), ClientID: target.ID, ConfiguratorID: p.ID}, RequestID: m.ID, Deadline: time.Now().Add(requestTTL)}
	u.sessions[s.SessionID] = s
	u.send(target.ID, "sessionRequested", map[string]any{"sessionId": s.SessionID, "displayName": p.Actor.DisplayName, "workspaceIdentity": p.Workspace, "expiresAt": s.Deadline.UnixMilli()})
	return nil
}

func (u *UseCase) acceptSession(p *participant, m entities.BridgeMessage) error {
	s := u.sessions[m.SessionID]
	if s == nil || s.Active || s.ClientID != p.ID || p.Role != "client" {
		return fmt.Errorf("Session request is not pending")
	}
	if !m.Accepted {
		u.end(s.SessionID, "Client declined the connection")
		return nil
	}
	s.Active = true
	s.Deadline = time.Now().Add(sessionTTL)
	u.send(s.ClientID, "sessionStarted", s.BridgeSession)
	u.result(s.ConfiguratorID, s.RequestID, s.BridgeSession, "")
	return nil
}

func (u *UseCase) requestCommand(p *participant, m entities.BridgeMessage) error {
	s := u.sessions[m.SessionID]
	if !p.Debug || !shared.CanWrite(p.AccessRole) || s == nil || !s.Active || s.ConfiguratorID != p.ID || m.ID == "" {
		return fmt.Errorf("Debug session is not available")
	}
	if m.Type == "runSimulation" && (strings.TrimSpace(m.Identity) == "" || !hashPattern.MatchString(m.ExpectedHash)) {
		return fmt.Errorf("Simulation identity and SHA-256 are required")
	}
	if (m.Type == "executeCommand" || m.Type == "setInspectionOptions") && (len(m.Data) == 0 || len(m.Data) > maxCommandBytes || !json.Valid(m.Data)) {
		return fmt.Errorf("Invalid command payload")
	}
	count := 0
	for _, pending := range u.commands {
		if pending.SessionID == s.SessionID {
			count++
		}
	}
	if count >= 8 {
		return fmt.Errorf("Pending command limit reached")
	}
	commandID := uuid.NewString()
	u.commands[commandID] = command{SessionID: s.SessionID, RequestID: m.ID, Deadline: time.Now().Add(requestTTL)}
	m.ID = commandID
	if m.Type != "executeCommand" && m.Type != "setInspectionOptions" {
		m.Data = nil
	}
	m.Error = ""
	if !u.delivery.Send(s.ClientID, m) {
		u.end(s.SessionID, "Client is unavailable")
	}
	return nil
}

// forwardClientEvent routes opaque JSON only from the approved client to its controller.
func (u *UseCase) forwardClientEvent(p *participant, m entities.BridgeMessage) error {
	s := u.sessions[m.SessionID]
	if p.Role != "client" || !p.Debug || s == nil || !s.Active || s.ClientID != p.ID {
		return fmt.Errorf("Event session is not available")
	}
	limit := maxEventBytes
	if m.Type == "inspectionSnapshot" {
		limit = maxInspectionBytes
	}
	if len(m.Data) == 0 || len(m.Data) > limit || !json.Valid(m.Data) {
		return fmt.Errorf("Invalid event payload")
	}
	if !u.delivery.Send(s.ConfiguratorID, entities.BridgeMessage{Type: m.Type, SessionID: s.SessionID, Data: m.Data}) {
		u.end(s.SessionID, "Configurator is unavailable")
	}
	return nil
}

// Leave is idempotent and releases sessions and response payload ownership.
func (u *UseCase) Leave(id string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	p := u.peers[id]
	if p == nil {
		return
	}
	delete(u.peers, id)
	for sessionID, s := range u.sessions {
		if s.ClientID == id || s.ConfiguratorID == id {
			u.end(sessionID, "Connection closed")
		}
	}
	u.broadcast(p.Workspace)
}

// Sweep removes pending requests even when neither endpoint sends further traffic.
func (u *UseCase) Sweep() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.expire(time.Now())
}

func (u *UseCase) expire(now time.Time) {
	for id, s := range u.sessions {
		if !s.Deadline.After(now) {
			u.end(id, "Session expired")
		}
	}
	for id, c := range u.commands {
		if !c.Deadline.After(now) {
			if s := u.sessions[c.SessionID]; s != nil {
				u.result(s.ConfiguratorID, c.RequestID, nil, "Client command timed out")
			}
			delete(u.commands, id)
		}
	}
}

func (u *UseCase) end(id, reason string) {
	s := u.sessions[id]
	if s == nil {
		return
	}
	delete(u.sessions, id)
	if !s.Active {
		u.result(s.ConfiguratorID, s.RequestID, nil, reason)
	}
	for commandID, c := range u.commands {
		if c.SessionID == id {
			u.result(s.ConfiguratorID, c.RequestID, nil, reason)
			delete(u.commands, commandID)
		}
	}
	m := entities.BridgeMessage{Type: "sessionEnded", SessionID: id}
	u.delivery.Send(s.ClientID, m)
	u.delivery.Send(s.ConfiguratorID, m)
}

func (u *UseCase) authorize(ctx context.Context, p participant) (entities.Actor, string, error) {
	if p.Principal.UserID == "" || !p.Principal.ExpiresAt.After(time.Now()) {
		return entities.Actor{}, "", fmt.Errorf("Authentication expired")
	}
	actor, err := u.access.BridgeUser(ctx, p.Principal.UserID, p.Principal.SessionID)
	if err != nil {
		return actor, "", fmt.Errorf("Authentication is not valid")
	}
	workspace, err := u.workspaces.GetWorkspace(ctx, p.Workspace)
	if err != nil || workspace == nil || !workspace.Active {
		return actor, "", fmt.Errorf("Workspace is not available")
	}
	platform, err := u.grants.IsPlatformAdmin(ctx, actor.ID)
	if err != nil {
		return actor, "", fmt.Errorf("Access check failed")
	}
	role, err := u.workspaces.WorkspaceRole(ctx, workspace.ID, actor.ID, platform)
	if err != nil || (role != "viewer" && !shared.CanWrite(role)) {
		return actor, "", fmt.Errorf("Workspace access denied")
	}
	return actor, role, nil
}

func (u *UseCase) broadcast(workspace string) {
	configurators := []entities.BridgeConfigurator{}
	clients := []entities.BridgeClient{}
	for _, p := range u.peers {
		if p.Workspace != workspace {
			continue
		}
		if p.Role == "configurator" {
			configurators = append(configurators, entities.BridgeConfigurator{InstanceID: p.ID, UserID: p.Actor.ID, DisplayName: p.Actor.DisplayName, Label: p.Label})
		} else if p.Debug {
			clients = append(clients, entities.BridgeClient{InstanceID: p.ID, Label: p.Label})
		}
	}
	sort.Slice(configurators, func(i, j int) bool { return configurators[i].InstanceID < configurators[j].InstanceID })
	sort.Slice(clients, func(i, j int) bool { return clients[i].InstanceID < clients[j].InstanceID })
	for _, p := range u.peers {
		if p.Workspace != workspace || p.Role != "configurator" {
			continue
		}
		u.send(p.ID, "configurators", configurators)
		visible := []entities.BridgeClient{}
		if p.Debug && shared.CanWrite(p.AccessRole) {
			visible = clients
		}
		u.send(p.ID, "clients", visible)
	}
}

func (u *UseCase) send(id, kind string, data any) {
	payload, _ := json.Marshal(data)
	u.delivery.Send(id, entities.BridgeMessage{Type: kind, Data: payload})
}

func (u *UseCase) result(id, requestID string, data any, reason string) {
	payload, _ := json.Marshal(data)
	u.delivery.Send(id, entities.BridgeMessage{Type: "result", ID: requestID, Data: payload, Error: reason})
}
