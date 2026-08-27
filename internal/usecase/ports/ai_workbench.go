package ports

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

var (
	ErrWorkbenchUnavailable = errors.New("ai workbench unavailable")
	ErrWorkbenchTimeout     = errors.New("ai workbench timeout")
	ErrWorkbenchBadGateway  = errors.New("ai workbench bad gateway")
	ErrWorkbenchNotFound    = errors.New("ai workbench resource not found")
	ErrWorkbenchConflict    = errors.New("ai workbench state conflict")
	ErrWorkbenchInvalid     = errors.New("ai workbench invalid request")
)

type WorkbenchActor struct {
	ID          string
	Username    string
	DisplayName string
}

type WorkbenchWorkspace struct {
	ID         string
	Name       string
	Generation string
	Head       int64
}

// WorkbenchProviderAccess exists only for the duration of one authorized run.
// It must never be persisted or exposed through public HTTP responses.
type WorkbenchProviderAccess struct {
	ConnectionID string
	BaseURL      string
	Credential   string
}

type WorkbenchRunRequest struct {
	RequestID      string
	Actor          WorkbenchActor
	Workspace      WorkbenchWorkspace
	ConversationID string
	Prompt         string
	Model          entities.AIModelSnapshot
	Snapshot       json.RawMessage
	SnapshotSHA256 string
	ProviderAccess WorkbenchProviderAccess
}

type ConnectedServiceInfo struct {
	Service string
	Version string
	Env     string
}

type ConnectedServiceInfoProvider interface {
	ServiceInfo(context.Context) (ConnectedServiceInfo, error)
}

type AIWorkbenchGateway interface {
	Capabilities(context.Context) ([]string, error)
	ListConversations(context.Context, string, string, bool, int, string) ([]entities.AIConversation, string, error)
	CreateConversation(context.Context, WorkbenchActor, WorkbenchWorkspace, entities.AIModelSnapshot) (*entities.AIConversation, error)
	ResetConversation(context.Context, WorkbenchActor, WorkbenchWorkspace, string, entities.AIModelSnapshot) (*entities.AIConversation, error)
	UpdateConversationModel(context.Context, string, string, string, entities.AIModelSnapshot) (*entities.AIConversation, error)
	ListMessages(context.Context, string, string, string, int, string) ([]entities.AIMessage, string, error)
	Run(context.Context, WorkbenchRunRequest, func(entities.AIRunEvent) error) error
}
