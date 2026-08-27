package workbench

import (
	"context"
	"errors"
	"io"
	"strconv"
	"sync"
	"time"

	workbenchpb "github.com/endge-lab/service-backend/internal/adapter/workbenchpb"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type Client struct {
	client         workbenchpb.WorkbenchServiceClient
	health         grpc_health_v1.HealthClient
	requestTimeout time.Duration
	healthTimeout  time.Duration
	healthCacheTTL time.Duration

	mu              sync.Mutex
	cachedAdapters  []string
	capabilitiesAt  time.Time
	capabilitiesErr error

	infoMu            sync.Mutex
	cachedServiceInfo ports.ConnectedServiceInfo
	serviceInfoAt     time.Time
	serviceInfoErr    error
}

const serviceName = "service_ai_workbench"

func NewClient(client workbenchpb.WorkbenchServiceClient, health grpc_health_v1.HealthClient, requestTimeout, healthTimeout, healthCacheTTL time.Duration) *Client {
	return &Client{client: client, health: health, requestTimeout: requestTimeout, healthTimeout: healthTimeout, healthCacheTTL: healthCacheTTL}
}

func (c *Client) ServiceInfo(ctx context.Context) (ports.ConnectedServiceInfo, error) {
	fallback := ports.ConnectedServiceInfo{Service: serviceName}
	if c.client == nil {
		return fallback, ports.ErrWorkbenchUnavailable
	}
	c.infoMu.Lock()
	defer c.infoMu.Unlock()
	if !c.serviceInfoAt.IsZero() && time.Since(c.serviceInfoAt) < c.healthCacheTTL {
		return c.cachedServiceInfo, c.serviceInfoErr
	}
	callCtx, cancel := context.WithTimeout(ctx, c.healthTimeout)
	defer cancel()
	response, err := c.client.GetServiceInfo(callCtx, &workbenchpb.GetServiceInfoRequest{})
	if err != nil {
		c.cacheServiceInfo(fallback, mapError(err))
		return c.cachedServiceInfo, c.serviceInfoErr
	}
	service := response.GetService()
	if service == "" {
		service = serviceName
	}
	c.cacheServiceInfo(ports.ConnectedServiceInfo{Service: service, Version: response.GetVersion(), Env: response.GetEnv()}, nil)
	return c.cachedServiceInfo, nil
}

func (c *Client) cacheServiceInfo(info ports.ConnectedServiceInfo, err error) {
	c.cachedServiceInfo = info
	c.serviceInfoErr = err
	c.serviceInfoAt = time.Now()
}

func (c *Client) Capabilities(ctx context.Context) ([]string, error) {
	if c.client == nil || c.health == nil {
		return nil, ports.ErrWorkbenchUnavailable
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.capabilitiesAt.IsZero() && time.Since(c.capabilitiesAt) < c.healthCacheTTL {
		return append([]string(nil), c.cachedAdapters...), c.capabilitiesErr
	}
	healthCtx, cancel := context.WithTimeout(ctx, c.healthTimeout)
	defer cancel()
	healthResponse, err := c.health.Check(healthCtx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil || healthResponse.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		c.cacheCapabilities(nil, mapError(err))
		return nil, c.capabilitiesErr
	}
	response, err := c.client.GetCapabilities(healthCtx, &workbenchpb.GetCapabilitiesRequest{})
	if err != nil {
		c.cacheCapabilities(nil, mapError(err))
		return nil, c.capabilitiesErr
	}
	c.cacheCapabilities(response.GetAdapters(), nil)
	return append([]string(nil), c.cachedAdapters...), nil
}

func (c *Client) cacheCapabilities(adapters []string, err error) {
	c.cachedAdapters = append([]string(nil), adapters...)
	c.capabilitiesErr = err
	c.capabilitiesAt = time.Now()
}

func (c *Client) ListConversations(ctx context.Context, actorID, workspaceID string, includeArchived bool, limit int, cursor string) ([]entities.AIConversation, string, error) {
	if c.client == nil {
		return nil, "", ports.ErrWorkbenchUnavailable
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	response, err := c.client.ListConversations(callCtx, &workbenchpb.ListConversationsRequest{
		ActorId: actorID, WorkspaceId: workspaceID, IncludeArchived: includeArchived, Limit: uint32(limit), Cursor: cursor,
	})
	if err != nil {
		return nil, "", mapError(err)
	}
	items := make([]entities.AIConversation, 0, len(response.GetItems()))
	for _, item := range response.GetItems() {
		items = append(items, conversationFromProto(item))
	}
	return items, response.GetNextCursor(), nil
}

func (c *Client) CreateConversation(ctx context.Context, actor ports.WorkbenchActor, workspace ports.WorkbenchWorkspace, model entities.AIModelSnapshot) (*entities.AIConversation, error) {
	if c.client == nil {
		return nil, ports.ErrWorkbenchUnavailable
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	response, err := c.client.CreateConversation(callCtx, &workbenchpb.CreateConversationRequest{
		Actor: actorToProto(actor), Workspace: workspaceToProto(workspace), Model: modelToProto(model),
	})
	if err != nil {
		return nil, mapError(err)
	}
	value := conversationFromProto(response.GetConversation())
	return &value, nil
}

func (c *Client) ResetConversation(ctx context.Context, actor ports.WorkbenchActor, workspace ports.WorkbenchWorkspace, currentID string, model entities.AIModelSnapshot) (*entities.AIConversation, error) {
	if c.client == nil {
		return nil, ports.ErrWorkbenchUnavailable
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	response, err := c.client.ResetConversation(callCtx, &workbenchpb.ResetConversationRequest{
		Actor: actorToProto(actor), Workspace: workspaceToProto(workspace), CurrentConversationId: currentID, Model: modelToProto(model),
	})
	if err != nil {
		return nil, mapError(err)
	}
	value := conversationFromProto(response.GetConversation())
	return &value, nil
}

func (c *Client) UpdateConversationModel(ctx context.Context, actorID, workspaceID, conversationID string, model entities.AIModelSnapshot) (*entities.AIConversation, error) {
	if c.client == nil {
		return nil, ports.ErrWorkbenchUnavailable
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	response, err := c.client.UpdateConversationModel(callCtx, &workbenchpb.UpdateConversationModelRequest{
		ActorId: actorID, WorkspaceId: workspaceID, ConversationId: conversationID, Model: modelToProto(model),
	})
	if err != nil {
		return nil, mapError(err)
	}
	value := conversationFromProto(response.GetConversation())
	return &value, nil
}

func (c *Client) ListMessages(ctx context.Context, actorID, workspaceID, conversationID string, limit int, cursor string) ([]entities.AIMessage, string, error) {
	if c.client == nil {
		return nil, "", ports.ErrWorkbenchUnavailable
	}
	callCtx, cancel := c.callContext(ctx)
	defer cancel()
	response, err := c.client.ListMessages(callCtx, &workbenchpb.ListMessagesRequest{
		ActorId: actorID, WorkspaceId: workspaceID, ConversationId: conversationID, Limit: uint32(limit), Cursor: cursor,
	})
	if err != nil {
		return nil, "", mapError(err)
	}
	items := make([]entities.AIMessage, 0, len(response.GetItems()))
	for _, item := range response.GetItems() {
		role := "user"
		if item.GetRole() == workbenchpb.MessageRole_MESSAGE_ROLE_ASSISTANT {
			role = "assistant"
		}
		items = append(items, entities.AIMessage{
			ID: item.GetId(), ConversationID: item.GetConversationId(), Role: role, Content: item.GetContent(),
			Sequence: item.GetSequence(), CreatedAt: item.GetCreatedAt().AsTime(),
		})
	}
	return items, response.GetNextCursor(), nil
}

func (c *Client) Run(ctx context.Context, request ports.WorkbenchRunRequest, emit func(entities.AIRunEvent) error) error {
	if c.client == nil {
		return ports.ErrWorkbenchUnavailable
	}
	stream, err := c.client.Run(ctx, &workbenchpb.RunRequest{
		RequestId: request.RequestID, Actor: actorToProto(request.Actor), Workspace: workspaceToProto(request.Workspace),
		ConversationId: request.ConversationID, Prompt: request.Prompt, Model: modelToProto(request.Model),
		Snapshot: &workbenchpb.WorkspaceSnapshot{Payload: request.Snapshot, Generation: request.Workspace.Generation, HeadRevisionId: formatHead(request.Workspace.Head), Sha256: request.SnapshotSHA256},
		ProviderAccess: &workbenchpb.ProviderAccess{
			ConnectionId: request.ProviderAccess.ConnectionID,
			BaseUrl:      request.ProviderAccess.BaseURL,
			Credential:   request.ProviderAccess.Credential,
		},
	})
	if err != nil {
		return mapError(err)
	}
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return mapError(err)
		}
		if err := emit(runEventFromProto(response)); err != nil {
			return err
		}
	}
}

func (c *Client) callContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.requestTimeout)
}

func actorToProto(value ports.WorkbenchActor) *workbenchpb.Actor {
	return &workbenchpb.Actor{Id: value.ID, Username: value.Username, DisplayName: value.DisplayName}
}

func workspaceToProto(value ports.WorkbenchWorkspace) *workbenchpb.Workspace {
	return &workbenchpb.Workspace{Id: value.ID, Name: value.Name}
}

func modelToProto(value entities.AIModelSnapshot) *workbenchpb.ModelSnapshot {
	return &workbenchpb.ModelSnapshot{ProfileId: value.ProfileID, ConnectionId: value.ConnectionID, Adapter: value.Adapter, ProviderModelId: value.ProviderModelID, DisplayName: value.DisplayName}
}

func conversationFromProto(value *workbenchpb.Conversation) entities.AIConversation {
	if value == nil {
		return entities.AIConversation{}
	}
	model := value.GetModel()
	return entities.AIConversation{
		ID: value.GetId(), WorkspaceID: value.GetWorkspaceId(), Model: entities.AIModelSnapshot{
			ProfileID: model.GetProfileId(), ConnectionID: model.GetConnectionId(), Adapter: model.GetAdapter(), ProviderModelID: model.GetProviderModelId(), DisplayName: model.GetDisplayName(),
		}, Archived: value.GetArchived(), MessageCount: value.GetMessageCount(), CreatedAt: value.GetCreatedAt().AsTime(), UpdatedAt: value.GetUpdatedAt().AsTime(),
	}
}

func runEventFromProto(value *workbenchpb.RunResponse) entities.AIRunEvent {
	eventType := "failed"
	switch value.GetType() {
	case workbenchpb.RunEventType_RUN_EVENT_TYPE_STARTED:
		eventType = "started"
	case workbenchpb.RunEventType_RUN_EVENT_TYPE_CONTENT_DELTA:
		eventType = "content_delta"
	case workbenchpb.RunEventType_RUN_EVENT_TYPE_COMPLETED:
		eventType = "completed"
	}
	return entities.AIRunEvent{Type: eventType, RunID: value.GetRunId(), MessageID: value.GetMessageId(), Delta: value.GetDelta(), ErrorCode: value.GetErrorCode(), ErrorMessage: value.GetErrorMessage(), CreatedAt: value.GetCreatedAt().AsTime()}
}

func mapError(err error) error {
	if err == nil {
		return ports.ErrWorkbenchUnavailable
	}
	switch status.Code(err) {
	case codes.InvalidArgument:
		return ports.ErrWorkbenchInvalid
	case codes.NotFound:
		return ports.ErrWorkbenchNotFound
	case codes.FailedPrecondition, codes.AlreadyExists, codes.Aborted:
		return ports.ErrWorkbenchConflict
	case codes.DeadlineExceeded:
		return ports.ErrWorkbenchTimeout
	case codes.Unavailable, codes.Unauthenticated:
		return ports.ErrWorkbenchUnavailable
	default:
		return ports.ErrWorkbenchBadGateway
	}
}

func formatHead(value int64) string {
	return strconv.FormatInt(value, 10)
}
