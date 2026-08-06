package releases

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type artifactReaderSpy struct {
	operation   string
	workspaceID string
	release     entities.Release
}

func (s *artifactReaderSpy) Read(_ context.Context, operation, workspaceID string, release entities.Release) (*entities.ReleaseArtifact, error) {
	s.operation = operation
	s.workspaceID = workspaceID
	s.release = release
	return &entities.ReleaseArtifact{ReleaseID: release.ID, WorkspaceID: workspaceID, Checksum: release.Checksum, Data: []byte(`{}`)}, nil
}

func TestGetArtifactDelegatesToReaderPort(t *testing.T) {
	reader := &artifactReaderSpy{}
	usecase := &UseCase{artifacts: reader}
	release := entities.Release{ID: "release-id", Checksum: "checksum"}
	ctx := entities.WithWorkspaceAccess(context.Background(), entities.WorkspaceAccess{Workspace: entities.Workspace{ID: "workspace-id"}})

	artifact, err := usecase.GetArtifact(ctx, release)
	if err != nil {
		t.Fatal(err)
	}
	if reader.operation != "export" || reader.workspaceID != "workspace-id" || reader.release != release {
		t.Fatalf("reader received operation=%q workspace=%q release=%#v", reader.operation, reader.workspaceID, reader.release)
	}
	if artifact.ReleaseID != release.ID {
		t.Fatalf("artifact=%#v", artifact)
	}
}
