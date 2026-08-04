package session

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"
)

// Result содержит проекцию текущей пользовательской сессии.
type Result struct {
	User          *entities.User       `json:"user"`
	PlatformAdmin bool                 `json:"platformAdmin"`
	Workspaces    []entities.Workspace `json:"workspaces"`
}

// UseCase координирует сценарии работы с текущей пользовательской сессией.
type UseCase struct{ workspaces *workspaces.UseCase }

// NewUseCase создаёт use case для работы с текущей пользовательской сессией.
func NewUseCase(workspaceUseCase *workspaces.UseCase) *UseCase {
	return &UseCase{workspaces: workspaceUseCase}
}

// Current возвращает текущую сессию пользователя и доступные рабочие пространства.
func (s *UseCase) Current(ctx context.Context) (*Result, error) {
	actor, _ := entities.CurrentActorFromContext(ctx)
	items, err := s.workspaces.List(ctx)
	if err != nil {
		return nil, err
	}
	return &Result{User: actor.User, PlatformAdmin: actor.PlatformAdmin, Workspaces: items}, nil
}
