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
	value, err := s.releaseMetadata(ctx, scope.Workspace.ID, identity)
	return value, shared.MapNotFound(err)
}

// GetArtifact читает большой JSON уже разрешённого release в текущем workspace.
func (s *UseCase) GetArtifact(ctx context.Context, release entities.Release) (*entities.ReleaseArtifact, error) {
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, err
	}
	artifact, err := s.artifacts.Read(ctx, "export", scope.Workspace.ID, release)
	return artifact, shared.MapNotFound(err)
}

func (s *UseCase) releaseMetadata(ctx context.Context, workspaceID, identity string) (*entities.Release, error) {
	if identity == "last" {
		return s.releases.GetLatestReleaseMetadata(ctx, workspaceID)
	}
	return s.releases.GetReleaseMetadata(ctx, workspaceID, identity)
}
