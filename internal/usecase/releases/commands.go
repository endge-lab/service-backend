package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

// Create создаёт неизменяемый релиз из выбранного коммита.
func (s *UseCase) Create(ctx context.Context, input CreateInput) (*entities.Release, error) {
	current, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	identity := strings.TrimSpace(input.Identity)
	if identity == "" {
		return nil, domainerrors.InvalidInput("identity_required", "identity is required")
	}
	if identity == "last" {
		return nil, domainerrors.InvalidInput("release_identity_reserved", "identity last is reserved for read-only lookup")
	}
	if len(identity) > 160 {
		return nil, domainerrors.InvalidInput("identity_too_long", "identity must not exceed 160 characters")
	}
	commit, err := s.commits.GetCommit(ctx, scope.Workspace.ID, input.SourceCommitID)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	bundle, err := s.portable.ExportWorkspace(ctx, scope.Workspace.ID, &commit.HeadSequence)
	if err != nil {
		return nil, err
	}
	var portable entities.PortableBundle
	if err = json.Unmarshal(bundle, &portable); err != nil {
		return nil, err
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = identity
	}
	sum := sha256.Sum256(bundle)
	value := entities.Release{ID: uuid.NewString(), WorkspaceID: scope.Workspace.ID, Identity: identity, DisplayName: displayName, Description: input.Description, SourceCommitID: commit.ID, HeadSequence: commit.HeadSequence, SchemaVersion: portable.SchemaVersion, Checksum: hex.EncodeToString(sum[:]), CreatedBy: entities.Actor{ID: current.User.ID}}
	created, err := s.releases.CreateRelease(ctx, value, bundle)
	return created, shared.MapConflict(err)
}
