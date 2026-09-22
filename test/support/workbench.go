//go:build e2e

package support

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	workbenchpb "github.com/endge-lab/service-backend/internal/adapter/workbenchpb"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcHealth "google.golang.org/grpc/health"
	grpcHealthPB "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FakeWorkbench is a deterministic in-memory implementation of the real
// Workbench gRPC contract. It is intentionally a test-only dependency.
type FakeWorkbench struct {
	workbenchpb.UnimplementedWorkbenchServiceServer
	mu            sync.Mutex
	server        *grpc.Server
	listener      net.Listener
	conversations map[string]*workbenchpb.Conversation
	messages      map[string][]*workbenchpb.Message
	lastRun       *workbenchpb.RunRequest
}

// NewFakeWorkbench starts a localhost gRPC Workbench and reports serving health.
func NewFakeWorkbench(t testing.TB) *FakeWorkbench {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start fake AI Workbench listener: %v", err)
	}
	fake := &FakeWorkbench{
		listener: listener, conversations: make(map[string]*workbenchpb.Conversation), messages: make(map[string][]*workbenchpb.Message),
	}
	fake.server = grpc.NewServer()
	workbenchpb.RegisterWorkbenchServiceServer(fake.server, fake)
	health := grpcHealth.NewServer()
	health.SetServingStatus("", grpcHealthPB.HealthCheckResponse_SERVING)
	grpcHealthPB.RegisterHealthServer(fake.server, health)
	go func() { _ = fake.server.Serve(listener) }()
	t.Cleanup(func() { fake.server.Stop() })
	return fake
}

// Target returns the plain-text localhost target expected by the test config.
func (f *FakeWorkbench) Target() string { return f.listener.Addr().String() }

// LastRun returns a copy of the most recent request received by Run.
func (f *FakeWorkbench) LastRun() *workbenchpb.RunRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastRun
}

func (f *FakeWorkbench) GetCapabilities(context.Context, *workbenchpb.GetCapabilitiesRequest) (*workbenchpb.GetCapabilitiesResponse, error) {
	return &workbenchpb.GetCapabilitiesResponse{Adapters: []string{"anthropic", "ollama"}}, nil
}

func (f *FakeWorkbench) ListConversations(_ context.Context, request *workbenchpb.ListConversationsRequest) (*workbenchpb.ListConversationsResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]*workbenchpb.Conversation, 0)
	for _, item := range f.conversations {
		if item.GetActorId() == request.GetActorId() && item.GetWorkspaceId() == request.GetWorkspaceId() && (request.GetIncludeArchived() || !item.GetArchived()) {
			items = append(items, item)
		}
	}
	return &workbenchpb.ListConversationsResponse{Items: items}, nil
}

func (f *FakeWorkbench) CreateConversation(_ context.Context, request *workbenchpb.CreateConversationRequest) (*workbenchpb.CreateConversationResponse, error) {
	if request.GetActor() == nil || request.GetWorkspace() == nil || request.GetModel() == nil {
		return nil, status.Error(codes.InvalidArgument, "actor, workspace and model are required")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	value := f.newConversation(request.GetActor().GetId(), request.GetWorkspace().GetId(), request.GetModel())
	return &workbenchpb.CreateConversationResponse{Conversation: value}, nil
}

func (f *FakeWorkbench) ResetConversation(_ context.Context, request *workbenchpb.ResetConversationRequest) (*workbenchpb.ResetConversationResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if request.GetCurrentConversationId() != "" {
		current, ok := f.conversations[request.GetCurrentConversationId()]
		if !ok || current.GetActorId() != request.GetActor().GetId() || current.GetWorkspaceId() != request.GetWorkspace().GetId() {
			return nil, status.Error(codes.NotFound, "conversation not found")
		}
		current.Archived = true
	}
	value := f.newConversation(request.GetActor().GetId(), request.GetWorkspace().GetId(), request.GetModel())
	return &workbenchpb.ResetConversationResponse{Conversation: value}, nil
}

func (f *FakeWorkbench) UpdateConversationModel(_ context.Context, request *workbenchpb.UpdateConversationModelRequest) (*workbenchpb.UpdateConversationModelResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	value, ok := f.conversations[request.GetConversationId()]
	if !ok || value.GetActorId() != request.GetActorId() || value.GetWorkspaceId() != request.GetWorkspaceId() {
		return nil, status.Error(codes.NotFound, "conversation not found")
	}
	if value.GetMessageCount() > 0 {
		return nil, status.Error(codes.FailedPrecondition, "conversation is not empty")
	}
	value.Model, value.UpdatedAt = request.GetModel(), timestamppb.New(time.Now().UTC())
	return &workbenchpb.UpdateConversationModelResponse{Conversation: value}, nil
}

func (f *FakeWorkbench) ListMessages(_ context.Context, request *workbenchpb.ListMessagesRequest) (*workbenchpb.ListMessagesResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	value, ok := f.conversations[request.GetConversationId()]
	if !ok || value.GetActorId() != request.GetActorId() || value.GetWorkspaceId() != request.GetWorkspaceId() {
		return nil, status.Error(codes.NotFound, "conversation not found")
	}
	return &workbenchpb.ListMessagesResponse{Items: f.messages[request.GetConversationId()]}, nil
}

func (f *FakeWorkbench) Run(request *workbenchpb.RunRequest, stream grpc.ServerStreamingServer[workbenchpb.RunResponse]) error {
	f.mu.Lock()
	conversation, ok := f.conversations[request.GetConversationId()]
	if !ok || conversation.GetActorId() != request.GetActor().GetId() || conversation.GetWorkspaceId() != request.GetWorkspace().GetId() {
		f.mu.Unlock()
		return status.Error(codes.NotFound, "conversation not found")
	}
	f.lastRun = request
	messageID := uuid.NewString()
	conversation.MessageCount++
	conversation.UpdatedAt = timestamppb.New(time.Now().UTC())
	f.messages[conversation.GetId()] = append(f.messages[conversation.GetId()], &workbenchpb.Message{
		Id: messageID, ConversationId: conversation.GetId(), Role: workbenchpb.MessageRole_MESSAGE_ROLE_ASSISTANT,
		Content: "fake answer", Sequence: conversation.GetMessageCount(), CreatedAt: timestamppb.New(time.Now().UTC()),
	})
	f.mu.Unlock()
	if err := stream.Send(&workbenchpb.RunResponse{Type: workbenchpb.RunEventType_RUN_EVENT_TYPE_STARTED, RunId: request.GetRequestId(), CreatedAt: timestamppb.Now()}); err != nil {
		return err
	}
	if err := stream.Send(&workbenchpb.RunResponse{Type: workbenchpb.RunEventType_RUN_EVENT_TYPE_CONTENT_DELTA, RunId: request.GetRequestId(), MessageId: messageID, Delta: "fake answer", CreatedAt: timestamppb.Now()}); err != nil {
		return err
	}
	return stream.Send(&workbenchpb.RunResponse{Type: workbenchpb.RunEventType_RUN_EVENT_TYPE_COMPLETED, RunId: request.GetRequestId(), MessageId: messageID, CreatedAt: timestamppb.Now()})
}

func (f *FakeWorkbench) GetServiceInfo(context.Context, *workbenchpb.GetServiceInfoRequest) (*workbenchpb.GetServiceInfoResponse, error) {
	return &workbenchpb.GetServiceInfoResponse{Service: "fake-workbench", Version: "test", Env: "test"}, nil
}

func (f *FakeWorkbench) newConversation(actorID, workspaceID string, model *workbenchpb.ModelSnapshot) *workbenchpb.Conversation {
	now := timestamppb.New(time.Now().UTC())
	value := &workbenchpb.Conversation{Id: uuid.NewString(), ActorId: actorID, WorkspaceId: workspaceID, Model: model, CreatedAt: now, UpdatedAt: now}
	f.conversations[value.GetId()] = value
	return value
}
