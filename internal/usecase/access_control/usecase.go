package access_control

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

type UseCase struct {
	repository ports.AccessControlRepository
	users      ports.UserRepository
	workspaces ports.WorkspaceRepository
	tx         ports.TxManager
}

type Page[T any] struct {
	Items      []T
	NextCursor string
}

type PutInput struct {
	UserID            string
	ScopeType         string
	WorkspaceIdentity string
	Role              string
}

type ListInput struct {
	ScopeType         string
	WorkspaceIdentity string
	UserID            string
	Query             string
	Cursor            string
	Limit             int
}

type BulkInput struct {
	UserID              string
	Role                string
	SelectionType       string
	WorkspaceIdentities []string
}

type BulkResult struct {
	Affected int `json:"affected"`
	Created  int `json:"created"`
	Updated  int `json:"updated"`
}

func NewUseCase(repository ports.AccessControlRepository, users ports.UserRepository, workspaces ports.WorkspaceRepository, tx ports.TxManager) *UseCase {
	return &UseCase{repository: repository, users: users, workspaces: workspaces, tx: tx}
}

// ResolveCurrentActor синхронизирует identity и вычисляет platform role из локальной БД.
func (s *UseCase) ResolveCurrentActor(ctx context.Context, input ports.UpsertCurrentUserInput, legacyPlatformAdmin bool) (*entities.User, bool, error) {
	hasAdmins, err := s.repository.HasPlatformAdmins(ctx)
	if err != nil {
		return nil, false, err
	}
	if hasAdmins && !legacyPlatformAdmin {
		user, upsertErr := s.users.UpsertCurrentUser(ctx, input)
		if upsertErr != nil {
			return nil, false, upsertErr
		}
		platform, roleErr := s.repository.IsPlatformAdmin(ctx, user.ID)
		return user, platform, roleErr
	}

	var user *entities.User
	var platform bool
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if err := s.repository.LockBootstrap(txctx); err != nil {
			return err
		}
		var err error
		user, err = s.users.UpsertCurrentUser(txctx, input)
		if err != nil {
			return err
		}
		if !user.Active {
			return nil
		}
		platform, err = s.repository.IsPlatformAdmin(txctx, user.ID)
		if err != nil {
			return err
		}
		if !platform {
			adminsExist, checkErr := s.repository.HasPlatformAdmins(txctx)
			if checkErr != nil {
				return checkErr
			}
			humanCount, countErr := s.repository.CountHumanUsers(txctx)
			if countErr != nil {
				return countErr
			}
			if legacyPlatformAdmin || (!adminsExist && humanCount == 1) {
				if _, _, err = s.repository.UpsertAccessGrant(txctx, ports.AccessGrantInput{UserID: user.ID, ScopeType: "platform", Role: "admin", ActorID: user.ID}); err != nil {
					return err
				}
				platform = true
			}
		}
		return nil
	})
	return user, platform, err
}

func (s *UseCase) SearchUsers(ctx context.Context, query, workspaceIdentity, cursor string, limit int) (Page[entities.AccessGrantUser], error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 2 {
		return Page[entities.AccessGrantUser]{}, domainerrors.InvalidInput("user_query_too_short", "q must contain at least 2 characters")
	}
	if err := s.requireGrantAdmin(ctx, workspaceIdentity); err != nil {
		return Page[entities.AccessGrantUser]{}, err
	}
	decoded, err := decodeCursor(cursor)
	if err != nil {
		return Page[entities.AccessGrantUser]{}, err
	}
	page, err := s.repository.SearchServiceUsers(ctx, ports.ServiceUserSearchInput{Query: query, Cursor: decoded, Limit: normalizedLimit(limit, 20, 50)})
	if err != nil {
		return Page[entities.AccessGrantUser]{}, err
	}
	return Page[entities.AccessGrantUser]{Items: page.Items, NextCursor: encodeCursor(page.Next)}, nil
}

func (s *UseCase) List(ctx context.Context, input ListInput) (Page[entities.AccessGrant], error) {
	input.ScopeType = strings.TrimSpace(input.ScopeType)
	if input.ScopeType != "platform" && input.ScopeType != "workspace" {
		return Page[entities.AccessGrant]{}, domainerrors.InvalidInput("access_scope_invalid", "scopeType must be platform or workspace")
	}
	actor, err := shared.Actor(ctx)
	if err != nil {
		return Page[entities.AccessGrant]{}, err
	}
	var workspaceID *string
	var userID *string
	if input.ScopeType == "platform" {
		if !actor.PlatformAdmin {
			return Page[entities.AccessGrant]{}, platformAdminRequired()
		}
		if strings.TrimSpace(input.UserID) != "" {
			if _, parseErr := uuid.Parse(strings.TrimSpace(input.UserID)); parseErr != nil {
				return Page[entities.AccessGrant]{}, domainerrors.InvalidInput("access_user_id_invalid", "userId must be UUID")
			}
			value := strings.TrimSpace(input.UserID)
			userID = &value
		}
	} else if strings.TrimSpace(input.UserID) != "" {
		if !actor.PlatformAdmin {
			return Page[entities.AccessGrant]{}, platformAdminRequired()
		}
		if _, parseErr := uuid.Parse(strings.TrimSpace(input.UserID)); parseErr != nil {
			return Page[entities.AccessGrant]{}, domainerrors.InvalidInput("access_user_id_invalid", "userId must be UUID")
		}
		value := strings.TrimSpace(input.UserID)
		userID = &value
	} else {
		workspace, authErr := s.requireWorkspaceAdmin(ctx, input.WorkspaceIdentity)
		if authErr != nil {
			return Page[entities.AccessGrant]{}, authErr
		}
		workspaceID = &workspace.ID
	}
	cursor, err := decodeCursor(input.Cursor)
	if err != nil {
		return Page[entities.AccessGrant]{}, err
	}
	page, err := s.repository.ListAccessGrants(ctx, ports.AccessGrantListInput{ScopeType: input.ScopeType, WorkspaceID: workspaceID, UserID: userID, Query: input.Query, Cursor: cursor, Limit: normalizedLimit(input.Limit, 50, 100)})
	if err != nil {
		return Page[entities.AccessGrant]{}, err
	}
	return Page[entities.AccessGrant]{Items: page.Items, NextCursor: encodeCursor(page.Next)}, nil
}

func (s *UseCase) Put(ctx context.Context, input PutInput) (*entities.AccessGrant, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(input.UserID)); err != nil {
		return nil, domainerrors.InvalidInput("access_user_id_invalid", "userId must be UUID")
	}
	input.ScopeType, input.Role = strings.TrimSpace(input.ScopeType), strings.TrimSpace(input.Role)
	grant := ports.AccessGrantInput{UserID: input.UserID, ScopeType: input.ScopeType, Role: input.Role, ActorID: actor.User.ID}
	if input.ScopeType == "platform" {
		if !actor.PlatformAdmin {
			return nil, platformAdminRequired()
		}
		if input.Role != "admin" {
			return nil, domainerrors.InvalidInput("platform_role_invalid", "platform scope only accepts admin")
		}
	} else if input.ScopeType == "workspace" {
		if !validWorkspaceRole(input.Role) {
			return nil, domainerrors.InvalidInput("workspace_role_invalid", "role must be viewer, editor or admin")
		}
		workspace, authErr := s.requireWorkspaceAdmin(ctx, input.WorkspaceIdentity)
		if authErr != nil {
			return nil, authErr
		}
		grant.WorkspaceID = &workspace.ID
	} else {
		return nil, domainerrors.InvalidInput("access_scope_invalid", "scopeType must be platform or workspace")
	}
	var value *entities.AccessGrant
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		var putErr error
		value, _, putErr = s.repository.UpsertAccessGrant(txctx, grant)
		return mapTargetNotFound(putErr)
	})
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *UseCase) Delete(ctx context.Context, id string) error {
	if _, parseErr := uuid.Parse(strings.TrimSpace(id)); parseErr != nil {
		return domainerrors.InvalidInput("access_grant_id_invalid", "id must be UUID")
	}
	actor, err := shared.Actor(ctx)
	if err != nil {
		return err
	}
	return s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		grant, getErr := s.repository.GetAccessGrant(txctx, id)
		if getErr != nil {
			return mapGrantNotFound(getErr)
		}
		if grant.ScopeType == "platform" {
			if !actor.PlatformAdmin {
				return platformAdminRequired()
			}
			if lockErr := s.repository.LockBootstrap(txctx); lockErr != nil {
				return lockErr
			}
			count, countErr := s.repository.CountPlatformAdmins(txctx)
			if countErr != nil {
				return countErr
			}
			if count <= 1 {
				return domainerrors.Conflict("last_platform_admin_required", "The last Platform Admin cannot be removed")
			}
		} else {
			if grant.WorkspaceIdentity == nil {
				return domainerrors.Internal("access_workspace_missing", "Access grant workspace is unavailable")
			}
			if _, authErr := s.requireWorkspaceAdmin(txctx, *grant.WorkspaceIdentity); authErr != nil {
				return authErr
			}
		}
		return mapGrantNotFound(s.repository.DeleteAccessGrant(txctx, id))
	})
}

func (s *UseCase) Bulk(ctx context.Context, input BulkInput) (BulkResult, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return BulkResult{}, err
	}
	if !actor.PlatformAdmin {
		return BulkResult{}, platformAdminRequired()
	}
	if _, err := uuid.Parse(strings.TrimSpace(input.UserID)); err != nil {
		return BulkResult{}, domainerrors.InvalidInput("access_user_id_invalid", "userId must be UUID")
	}
	if !validWorkspaceRole(input.Role) {
		return BulkResult{}, domainerrors.InvalidInput("workspace_role_invalid", "role must be viewer, editor or admin")
	}
	workspaces, err := s.workspaces.ListWorkspaces(ctx, actor.User.ID, true)
	if err != nil {
		return BulkResult{}, err
	}
	selected := map[string]bool{}
	if input.SelectionType == "selected" {
		for _, identity := range input.WorkspaceIdentities {
			selected[strings.TrimSpace(identity)] = true
		}
		if len(selected) == 0 {
			return BulkResult{}, domainerrors.InvalidInput("workspace_selection_required", "workspaceIdentities are required")
		}
	} else if input.SelectionType != "all-active" {
		return BulkResult{}, domainerrors.InvalidInput("workspace_selection_invalid", "selection.type must be all-active or selected")
	}
	result := BulkResult{}
	seen := map[string]bool{}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		for _, workspace := range workspaces {
			if !workspace.Active || (input.SelectionType == "selected" && !selected[workspace.Identity]) {
				continue
			}
			seen[workspace.Identity] = true
			workspaceID := workspace.ID
			_, created, putErr := s.repository.UpsertAccessGrant(txctx, ports.AccessGrantInput{UserID: input.UserID, ScopeType: "workspace", WorkspaceID: &workspaceID, Role: input.Role, ActorID: actor.User.ID})
			if putErr != nil {
				return mapTargetNotFound(putErr)
			}
			result.Affected++
			if created {
				result.Created++
			} else {
				result.Updated++
			}
		}
		if input.SelectionType == "selected" && len(seen) != len(selected) {
			return domainerrors.NotFound("workspace_not_found", "One or more selected Workspaces were not found or are inactive")
		}
		return nil
	})
	return result, err
}

func (s *UseCase) requireGrantAdmin(ctx context.Context, workspaceIdentity string) error {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return err
	}
	if actor.PlatformAdmin {
		return nil
	}
	_, err = s.requireWorkspaceAdmin(ctx, workspaceIdentity)
	return err
}

func (s *UseCase) requireWorkspaceAdmin(ctx context.Context, identity string) (*entities.Workspace, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return nil, domainerrors.InvalidInput("workspace_identity_required", "workspaceIdentity is required")
	}
	workspace, err := s.workspaces.GetWorkspace(ctx, identity)
	if err != nil {
		if err == ports.ErrNotFound {
			return nil, domainerrors.NotFound("workspace_not_found", "Workspace not found")
		}
		return nil, err
	}
	role, err := s.workspaces.WorkspaceRole(ctx, workspace.ID, actor.User.ID, actor.PlatformAdmin)
	if err != nil {
		return nil, err
	}
	if role != "admin" && role != "platform_admin" {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	return workspace, nil
}

func validWorkspaceRole(role string) bool {
	return role == "viewer" || role == "editor" || role == "admin"
}
func platformAdminRequired() error {
	return domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required")
}

func normalizedLimit(value, fallback, maximum int) int {
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func encodeCursor(cursor *entities.AccessGrantCursor) string {
	if cursor == nil {
		return ""
	}
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(raw string) (*entities.AccessGrantCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, domainerrors.InvalidInput("cursor_invalid", "cursor is invalid")
	}
	cursor := &entities.AccessGrantCursor{}
	if json.Unmarshal(value, cursor) != nil || strings.TrimSpace(cursor.ID) == "" {
		return nil, domainerrors.InvalidInput("cursor_invalid", "cursor is invalid")
	}
	if _, err := uuid.Parse(cursor.ID); err != nil {
		return nil, domainerrors.InvalidInput("cursor_invalid", "cursor is invalid")
	}
	return cursor, nil
}

func mapTargetNotFound(err error) error {
	if errors.Is(err, ports.ErrNotFound) {
		return domainerrors.NotFound("access_target_not_found", "User or Workspace not found")
	}
	return err
}

func mapGrantNotFound(err error) error {
	if errors.Is(err, ports.ErrNotFound) {
		return domainerrors.NotFound("access_grant_not_found", "Access grant not found")
	}
	return err
}
