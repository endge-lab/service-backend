package releases

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

// List возвращает релизы рабочего пространства.
func (s *UseCase) List(ctx context.Context) ([]entities.Release, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	return s.releases.ListReleases(ctx, scope.Workspace.ID)
}

// Get возвращает релиз рабочего пространства по идентификатору.
func (s *UseCase) Get(ctx context.Context, identity string) (*entities.Release, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	value, err := s.releases.GetRelease(ctx, scope.Workspace.ID, identity)
	return value, shared.MapNotFound(err)
}
