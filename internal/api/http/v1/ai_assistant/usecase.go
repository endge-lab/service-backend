package ai_assistant

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/ai_assistant"
)

type UseCase interface {
	Capabilities(context.Context) (resourceusecase.Capabilities, error)
	ListConversations(context.Context, bool, int, string) (resourceusecase.ConversationPage, error)
	CreateConversation(context.Context, string) (*entities.AIConversation, error)
	ResetConversation(context.Context, string, string) (*entities.AIConversation, error)
	UpdateConversationModel(context.Context, string, string) (*entities.AIConversation, error)
	ListMessages(context.Context, string, int, string) (resourceusecase.MessagePage, error)
	PrepareRun(context.Context, resourceusecase.RunCommand) (resourceusecase.PreparedRun, error)
	RunPrepared(context.Context, resourceusecase.PreparedRun, func(entities.AIRunEvent) error) error
}

func BindUseCase(usecase *resourceusecase.UseCase) UseCase { return usecase }
