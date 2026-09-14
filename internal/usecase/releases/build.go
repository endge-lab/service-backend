package releases

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

const MaxReleaseBuildBytes = 15 * 1024 * 1024
const maxDecodedBuildBytes = 256 * 1024 * 1024

type CreateFromBuildInput struct {
	CreateInput
	WorkspaceID   string                        `json:"workspaceId"`
	Generation    string                        `json:"generation"`
	HeadSequence  int64                         `json:"headSequence"`
	CommitMessage string                        `json:"commitMessage"`
	Metadata      entities.ReleaseBuildMetadata `json:"buildMetadata"`
	Data          []byte                        `json:"-"`
}

// CreateFromBuild publishes the snapshot, optional commit and build in one transaction.
func (s *UseCase) CreateFromBuild(ctx context.Context, input CreateFromBuildInput) (result *entities.Release, err error) {
	_, scope, err := shared.WriteContext(ctx)
	if err != nil {
		return nil, err
	}
	if input.WorkspaceID != scope.Workspace.ID || input.Generation == "" || input.HeadSequence < 0 {
		return nil, domainerrors.InvalidInput("build_source_invalid", "Build belongs to another Workspace or has no source revision")
	}
	metadata, err := validateReleaseBuild(input.Data, input.Metadata)
	if err != nil {
		return nil, err
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.builds.LockWorkspaceSnapshot(txctx, scope.Workspace.ID); err != nil {
			return err
		}
		workspace, err := s.builds.GetWorkspace(txctx, scope.Workspace.Identity)
		if err != nil {
			return err
		}
		if workspace.Generation != input.Generation {
			return domainerrors.Conflict("build_source_changed", "Workspace was replaced after build; rebuild before publishing")
		}
		scope.Workspace = *workspace
		txctx = entities.WithWorkspaceAccess(txctx, scope)
		sourceID := input.SourceCommitID
		if sourceID == "" {
			latest, err := s.commits.LatestCommit(txctx, workspace.ID)
			if err != nil {
				return err
			}
			if latest.HeadSequence == input.HeadSequence {
				sourceID = latest.ID
			} else {
				if workspace.HeadSequence != input.HeadSequence {
					return domainerrors.Conflict("build_source_changed", "Saved model changed after build; rebuild before creating a commit")
				}
				if strings.TrimSpace(input.CommitMessage) == "" {
					return domainerrors.InvalidInput("commit_message_required", "Enter a commit message for the built model")
				}
				commit, err := s.commitWriter.Create(txctx, input.CommitMessage, entities.RevisionPolicyPreserve, input.HeadSequence)
				if err != nil {
					return err
				}
				sourceID = commit.ID
			}
		}
		commit, err := s.commits.GetCommit(txctx, workspace.ID, sourceID)
		if err != nil {
			return shared.MapNotFound(err)
		}
		if commit.HeadSequence != input.HeadSequence {
			return domainerrors.Conflict("build_commit_mismatch", "Commit and Bundle have different source revisions")
		}
		create := input.CreateInput
		create.SourceCommitID = sourceID
		result, err = s.Create(txctx, create)
		if err != nil {
			return err
		}
		if err = s.builds.StoreReleaseBuild(txctx, workspace.ID, result.ID, input.Data, *metadata); err != nil {
			return err
		}
		result.BuildMetadata = metadata
		return nil
	})
	if err != nil {
		return nil, shared.MapConflict(err)
	}
	return result, nil
}

func (s *UseCase) GetBuild(ctx context.Context, identity string) ([]byte, *entities.Release, error) {
	value, err := s.Get(ctx, identity)
	if err != nil {
		return nil, nil, err
	}
	scope, err := shared.Access(ctx)
	if err != nil {
		return nil, nil, err
	}
	data, err := s.builds.GetReleaseBuild(ctx, scope.Workspace.ID, value.ID)
	return data, value, shared.MapNotFound(err)
}

func validateReleaseBuild(data []byte, metadata entities.ReleaseBuildMetadata) (*entities.ReleaseBuildMetadata, error) {
	invalid := func(message string) (*entities.ReleaseBuildMetadata, error) {
		return nil, domainerrors.InvalidInput("release_build_invalid", message)
	}
	if len(data) == 0 || len(data) > MaxReleaseBuildBytes {
		return invalid("Compressed Bundle must not exceed 15 MiB")
	}
	if metadata.Version != 1 || metadata.Runtime != "ts-browser" || metadata.Scope != "complete-model" || metadata.ContextMode != "effective-context" || metadata.FileFormat != "gzip" {
		return invalid("Unsupported build metadata")
	}
	if metadata.Profile != nil && (metadata.Profile.Identity == "" || len(metadata.Profile.Identity) > 160 || len(metadata.Profile.DisplayName) > 255 || metadata.Profile.Revision < 1) {
		return invalid("Invalid build profile metadata")
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return invalid("Expected a Gzip Bundle")
	}
	decoded, err := io.ReadAll(io.LimitReader(reader, maxDecodedBuildBytes+1))
	closeErr := reader.Close()
	if err != nil || closeErr != nil || len(decoded) > maxDecodedBuildBytes {
		return invalid("Damaged or oversized Bundle")
	}
	var envelope struct {
		Format     string          `json:"format"`
		Version    int             `json:"version"`
		Inspection json.RawMessage `json:"inspection"`
		Bundle     *struct {
			Version         int                        `json:"version"`
			ProgramID       string                     `json:"programId"`
			CompilerVersion string                     `json:"compilerVersion"`
			Context         json.RawMessage            `json:"context"`
			Artifacts       map[string]json.RawMessage `json:"artifacts"`
			Catalog         json.RawMessage            `json:"catalog"`
		} `json:"bundle"`
	}
	if json.Unmarshal(decoded, &envelope) != nil || envelope.Format != "endge-bundle" || envelope.Version != 1 || envelope.Bundle == nil || len(envelope.Inspection) != 0 {
		return invalid("Release requires one program without an inspection recording")
	}
	program := envelope.Bundle
	if program.Version != 1 || program.ProgramID == "" || program.CompilerVersion == "" || program.Artifacts == nil || len(program.Context) == 0 || len(program.Catalog) == 0 {
		return invalid("Incomplete execution program")
	}
	if metadata.ProgramID != program.ProgramID || metadata.CompilerVersion != program.CompilerVersion {
		return invalid("Build metadata does not match Bundle")
	}
	metadata.Context = program.Context
	metadata.SizeBytes = int64(len(data))
	sum := sha256.Sum256(data)
	metadata.Checksum = hex.EncodeToString(sum[:])
	return &metadata, nil
}
