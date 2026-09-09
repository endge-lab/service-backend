package session

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/access_control"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"
)

// Result содержит проекцию текущей пользовательской сессии.
type Result struct {
	AccessManagement entities.AccessManagement
	User             *entities.User             `json:"user"`
	PlatformAdmin    bool                       `json:"platformAdmin"`
	Workspaces       []entities.WorkspaceAccess `json:"workspaces"`
}

// UseCase координирует сценарии работы с текущей пользовательской сессией.
type UseCase struct {
	workspaces *workspaces.UseCase
	access     *access_control.UseCase
}

// NewUseCase создаёт use case для работы с текущей пользовательской сессией.
func NewUseCase(workspaceUseCase *workspaces.UseCase, access *access_control.UseCase) *UseCase {
	return &UseCase{workspaces: workspaceUseCase, access: access}
}

// Current возвращает текущую сессию пользователя и доступные рабочие пространства.
func (s *UseCase) Current(ctx context.Context) (*Result, error) {
	actor, _ := entities.CurrentActorFromContext(ctx)
	items, err := s.workspaces.ListAccess(ctx)
	if err != nil {
		return nil, err
	}
	management, err := s.access.Management(ctx, actor.User.ID)
	if err != nil {
		return nil, err
	}
	return &Result{AccessManagement: management, User: actor.User, PlatformAdmin: actor.PlatformAdmin, Workspaces: items}, nil
}
