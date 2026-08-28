package ai_assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ai_catalog"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/endge-lab/service-backend/internal/usecase/workspace_state"
	"github.com/google/uuid"
)

type Capabilities struct {
	Available bool                      `json:"available"`
	CanView   bool                      `json:"canView"`
	CanRun    bool                      `json:"canRun"`
	Reason    string                    `json:"reason,omitempty"`
	Adapters  []string                  `json:"adapters"`
	Models    []entities.AIModelProfile `json:"models"`
}

type ConversationPage struct {
	Items      []entities.AIConversation
	NextCursor string
}

type MessagePage struct {
	Items             []entities.AIMessage
	NextCursor        string
	OpenClarification *entities.AIClarification
}

type RunCommand struct {
	RequestID              string
	ConversationID         string
	ModelProfileID         string
	Prompt                 string
	InteractionID          string
	ReplyToClarificationID string
	SelectedCandidateID    string
}

// PreparedRun contains the immutable input assembled while the HTTP request is
// still inside its authorization and database boundaries. The Workbench stream
// starts only after those short-lived operations have completed.
type PreparedRun struct {
	request ports.WorkbenchRunRequest
}

type UseCase struct {
	catalog   *ai_catalog.UseCase
	workspace *workspace_state.Coordinator
	workbench ports.AIWorkbenchGateway
}

func NewUseCase(catalog *ai_catalog.UseCase, workspace *workspace_state.Coordinator, workbench ports.AIWorkbenchGateway) *UseCase {
	return &UseCase{catalog: catalog, workspace: workspace, workbench: workbench}
}

func (u *UseCase) Capabilities(ctx context.Context) (Capabilities, error) {
	if _, _, err := chatContext(ctx); err != nil {
		return Capabilities{}, err
	}
	models, err := u.catalog.ListModels(ctx, true)
	if err != nil {
		return Capabilities{}, err
	}
	adapters, err := u.workbench.Capabilities(ctx)
	if err != nil {
		return Capabilities{CanView: true, Available: false, CanRun: false, Reason: "workbench_unavailable", Models: models, Adapters: []string{}}, nil
	}
	canRun := len(models) > 0
	reason := ""
	if !canRun {
		reason = "models_not_configured"
	}
	return Capabilities{Available: true, CanView: true, CanRun: canRun, Reason: reason, Models: models, Adapters: adapters}, nil
}

func (u *UseCase) ListConversations(ctx context.Context, includeArchived bool, limit int, cursor string) (ConversationPage, error) {
	actor, workspace, err := chatContext(ctx)
	if err != nil {
		return ConversationPage{}, err
	}
	items, next, err := u.workbench.ListConversations(ctx, actor.User.ID, workspace.Workspace.ID, includeArchived, limit, cursor)
	return ConversationPage{Items: items, NextCursor: next}, mapWorkbenchError(err)
}

func (u *UseCase) CreateConversation(ctx context.Context, modelProfileID string) (*entities.AIConversation, error) {
	actor, workspace, err := chatContext(ctx)
	if err != nil {
		return nil, err
	}
	model, err := u.catalog.ResolveEnabledModel(ctx, modelProfileID)
	if err != nil {
		return nil, err
	}
	value, err := u.workbench.CreateConversation(ctx, actorProjection(actor), workspaceProjection(workspace), modelSnapshot(*model))
	return value, mapWorkbenchError(err)
}

func (u *UseCase) ResetConversation(ctx context.Context, currentID, modelProfileID string) (*entities.AIConversation, error) {
	actor, workspace, err := chatContext(ctx)
	if err != nil {
		return nil, err
	}
	model, err := u.catalog.ResolveEnabledModel(ctx, modelProfileID)
	if err != nil {
		return nil, err
	}
	value, err := u.workbench.ResetConversation(ctx, actorProjection(actor), workspaceProjection(workspace), currentID, modelSnapshot(*model))
	return value, mapWorkbenchError(err)
}

func (u *UseCase) UpdateConversationModel(ctx context.Context, conversationID, modelProfileID string) (*entities.AIConversation, error) {
	actor, workspace, err := chatContext(ctx)
	if err != nil {
		return nil, err
	}
	model, err := u.catalog.ResolveEnabledModel(ctx, modelProfileID)
	if err != nil {
		return nil, err
	}
	value, err := u.workbench.UpdateConversationModel(ctx, actor.User.ID, workspace.Workspace.ID, conversationID, modelSnapshot(*model))
	return value, mapWorkbenchError(err)
}

func (u *UseCase) ListMessages(ctx context.Context, conversationID string, limit int, cursor string) (MessagePage, error) {
	actor, workspace, err := chatContext(ctx)
	if err != nil {
		return MessagePage{}, err
	}
	items, next, clarification, err := u.workbench.ListMessages(ctx, actor.User.ID, workspace.Workspace.ID, conversationID, limit, cursor)
	return MessagePage{Items: items, NextCursor: next, OpenClarification: clarification}, mapWorkbenchError(err)
}

func (u *UseCase) PrepareRun(ctx context.Context, command RunCommand) (PreparedRun, error) {
	actor, workspace, err := chatContext(ctx)
	if err != nil {
		return PreparedRun{}, err
	}
	if _, err := uuid.Parse(command.RequestID); err != nil || strings.TrimSpace(command.Prompt) == "" {
		return PreparedRun{}, domainerrors.InvalidInput("ai.run_invalid", "requestId and prompt are required")
	}
	if (command.ReplyToClarificationID != "" && command.InteractionID == "") ||
		(command.SelectedCandidateID != "" && (command.InteractionID == "" || command.ReplyToClarificationID == "")) {
		return PreparedRun{}, domainerrors.InvalidInput("ai.clarification_linkage_invalid", "clarification linkage is incomplete")
	}
	if (command.InteractionID != "" && !validUUID(command.InteractionID)) ||
		(command.ReplyToClarificationID != "" && !validUUID(command.ReplyToClarificationID)) {
		return PreparedRun{}, domainerrors.InvalidInput("ai.clarification_linkage_invalid", "clarification linkage is invalid")
	}
	model, err := u.catalog.ResolveEnabledModel(ctx, command.ModelProfileID)
	if err != nil {
		return PreparedRun{}, err
	}
	providerAccess, err := u.catalog.ResolveProviderAccess(ctx, model.ConnectionID)
	if err != nil {
		return PreparedRun{}, err
	}
	snapshot, err := u.workspace.ExportLive(ctx)
	if err != nil {
		return PreparedRun{}, err
	}
	digest := sha256.Sum256(snapshot)
	return PreparedRun{request: ports.WorkbenchRunRequest{
		RequestID: command.RequestID, Actor: actorProjection(actor), Workspace: workspaceProjection(workspace),
		ConversationID: command.ConversationID, Prompt: strings.TrimSpace(command.Prompt), Model: modelSnapshot(*model),
		Snapshot: snapshot, SnapshotSHA256: hex.EncodeToString(digest[:]),
		InteractionID: command.InteractionID, ReplyToClarificationID: command.ReplyToClarificationID,
		SelectedCandidateID: command.SelectedCandidateID,
		ProviderAccess: ports.WorkbenchProviderAccess{
			ConnectionID: providerAccess.ConnectionID,
			BaseURL:      providerAccess.BaseURL,
			Credential:   providerAccess.Credential,
		},
	}}, nil
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func (u *UseCase) RunPrepared(ctx context.Context, prepared PreparedRun, emit func(entities.AIRunEvent) error) error {
	return mapWorkbenchError(u.workbench.Run(ctx, prepared.request, emit))
}

func chatContext(ctx context.Context) (entities.CurrentActor, entities.WorkspaceAccess, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return actor, entities.WorkspaceAccess{}, err
	}
	workspace, err := shared.Access(ctx)
	if err != nil {
		return actor, workspace, err
	}
	switch workspace.Role {
	case "viewer", "editor", "admin", "platform_admin":
		return actor, workspace, nil
	default:
		return actor, workspace, domainerrors.Forbidden("ai.viewer_required", "Workspace Viewer role is required")
	}
}

func actorProjection(actor entities.CurrentActor) ports.WorkbenchActor {
	return ports.WorkbenchActor{ID: actor.User.ID, Username: actor.User.Username, DisplayName: actor.User.DisplayName}
}

func workspaceProjection(access entities.WorkspaceAccess) ports.WorkbenchWorkspace {
	return ports.WorkbenchWorkspace{
		ID: access.Workspace.ID, Name: access.Workspace.DisplayName, Generation: access.Workspace.Generation, Head: access.Workspace.HeadSequence,
	}
}

func modelSnapshot(model entities.AIModelProfile) entities.AIModelSnapshot {
	return entities.AIModelSnapshot{
		ProfileID: model.ID, ConnectionID: model.ConnectionID, Adapter: model.Adapter,
		ProviderModelID: model.ProviderModelID, DisplayName: model.DisplayName,
	}
}

func mapWorkbenchError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ports.ErrWorkbenchUnavailable):
		return domainerrors.Wrap(err, "ai.workbench_unavailable", "AI Workbench is unavailable", 503)
	case errors.Is(err, ports.ErrWorkbenchTimeout):
		return domainerrors.Wrap(err, "ai.workbench_timeout", "AI Workbench timed out", 504)
	case errors.Is(err, ports.ErrWorkbenchBadGateway):
		return domainerrors.Wrap(err, "ai.workbench_bad_gateway", "AI Workbench returned an invalid response", 502)
	case errors.Is(err, ports.ErrWorkbenchNotFound):
		return domainerrors.NotFound("ai.conversation_not_found", "AI conversation not found")
	case errors.Is(err, ports.ErrWorkbenchConflict):
		return domainerrors.Conflict("ai.conversation_conflict", "AI conversation state does not allow this operation")
	case errors.Is(err, ports.ErrWorkbenchInvalid):
		return domainerrors.InvalidInput("ai.workbench_request_invalid", "AI Workbench request is invalid")
	default:
		return err
	}
}
