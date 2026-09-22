package access_control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

func (s *UseCase) resolveExternalActor(ctx context.Context, input ports.UpsertCurrentUserInput, snapshot *entities.ExternalAccessSnapshot) (*entities.User, bool, error) {
	if snapshot == nil || snapshot.ConfigVersion != s.policy.Version() || input.ProviderID != s.policy.Provider() ||
		snapshot.IssuedAt.IsZero() || snapshot.TokenHash == "" || !snapshot.ExpiresAt.After(time.Now()) {
		return nil, false, domainerrors.Forbidden("external_access_invalid", "Verified current external access is required")
	}
	var user *entities.User
	var platform bool
	err := s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		var err error
		// PostgreSQL upsert locks this identity's row until commit, serializing all sessions and bearer tokens.
		user, err = s.users.UpsertCurrentUser(txctx, input)
		if err != nil || !user.Active {
			return err
		}
		if !snapshot.ExpiresAt.After(time.Now()) {
			return domainerrors.Forbidden("external_access_expired", "Access token expired while preparing access")
		}
		grants := make([]ports.AccessGrantInput, 0, len(snapshot.Grants))
		for _, mapped := range snapshot.Grants {
			grant := ports.AccessGrantInput{UserID: user.ID, ActorID: user.ID, ScopeType: mapped.Scope, Role: mapped.Role}
			if mapped.Scope == entities.AccessScopePlatform {
				platform = true
			} else {
				workspace, err := s.workspaces.GetWorkspace(txctx, mapped.Workspace)
				if err != nil {
					return err
				}
				if !workspace.Active {
					return domainerrors.Forbidden("external_access_workspace_inactive", "External access workspace is inactive")
				}
				grant.WorkspaceID = &workspace.ID
			}
			grants = append(grants, grant)
		}
		encoded, err := json.Marshal(snapshot.Grants)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(encoded)
		state := entities.ExternalAccessState{ConfigVersion: snapshot.ConfigVersion, TokenHash: snapshot.TokenHash,
			IssuedAt: snapshot.IssuedAt, GrantsHash: hex.EncodeToString(digest[:])}
		previous, err := s.repository.GetExternalAccessState(txctx, user.ID)
		if err != nil {
			return err
		}
		if previous != nil {
			if state.IssuedAt.Before(previous.IssuedAt) {
				return domainerrors.Forbidden("external_access_stale", "A newer access token has already been applied; refresh the session")
			}
			if state.ConfigVersion == previous.ConfigVersion && state.IssuedAt.Equal(previous.IssuedAt) && state.TokenHash != previous.TokenHash && state.GrantsHash != previous.GrantsHash {
				return domainerrors.Forbidden("external_access_ambiguous", "Access tokens with equal iat disagree; obtain a newer token")
			}
			if state.ConfigVersion == previous.ConfigVersion && state.IssuedAt.Equal(previous.IssuedAt) && state.GrantsHash == previous.GrantsHash {
				return nil
			}
		}
		return s.repository.ReplaceExternalAccess(txctx, user.ID, grants, state)
	})
	return user, platform, err
}

// Management returns only the current user's synchronization metadata.
func (s *UseCase) Management(ctx context.Context, userID string) (entities.AccessManagement, error) {
	result := s.policy.Management()
	if !s.policy.External() {
		return result, nil
	}
	state, err := s.repository.GetExternalAccessState(ctx, userID)
	if err != nil {
		return result, err
	}
	if state != nil {
		result.LastSynchronizedAt = &state.LastSynchronizedAt
	}
	return result, nil
}
