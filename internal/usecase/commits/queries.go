package commits

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// List возвращает коммиты рабочего пространства.
func (s *UseCase) List(ctx context.Context) ([]entities.Commit, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	return s.repository.ListCommits(ctx, scope.Workspace.ID)
}

// Get возвращает коммит рабочего пространства по идентификатору.
func (s *UseCase) Get(ctx context.Context, id string) (*entities.Commit, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	value, err := s.repository.GetCommit(ctx, scope.Workspace.ID, id)
	return value, shared.MapNotFound(err)
}
