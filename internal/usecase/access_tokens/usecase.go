// Package access_tokens выпускает, перечисляет, отзывает и проверяет персональные
// токены доступа агентов (PAT) с правами владельца в пределах выбранной области.
package access_tokens

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

const (
	// LastUsedInterval ограничивает частоту записи last_used_at: не чаще раза в минуту на токен.
	LastUsedInterval = time.Minute
	// minimumLifetime отсекает токены, которые истекут раньше, чем агент успеет их применить.
	minimumLifetime = time.Minute
	maxNameRunes    = 120
	maxDescription  = 1000
)

type UseCase struct {
	tokens     ports.AccessTokenRepository
	workspaces ports.WorkspaceRepository
	maxTTL     time.Duration
	now        func() time.Time
}

// CreateInput — параметры выпуска; ExpiresAt обязателен и не больше now+maxTTL.
type CreateInput struct {
	WorkspaceIdentity string
	AllWorkspaces     bool
	ManageWorkspaces  bool
	Name              string
	Description       string
	Scope             string
	CanCommit         bool
	CanDelete         bool
	ExpiresAt         time.Time
}

// Created содержит секрет, который возвращается клиенту единственный раз.
type Created struct {
	Token  entities.AccessToken
	Secret string
}

// Authenticated — результат проверки секрета для auth middleware.
type Authenticated struct {
	Principal entities.AccessTokenPrincipal
	Owner     entities.User
}

func NewUseCase(tokens ports.AccessTokenRepository, workspaces ports.WorkspaceRepository, maxTTL time.Duration) *UseCase {
	return &UseCase{tokens: tokens, workspaces: workspaces, maxTTL: maxTTL, now: time.Now}
}

// MaxTTL возвращает действующий предел срока для подсказки клиенту.
func (u *UseCase) MaxTTL() time.Duration { return u.maxTTL }

// Create выпускает токен текущему пользователю. Только cookie/JWT сессия может выпускать токены.
func (u *UseCase) Create(ctx context.Context, input CreateInput) (*Created, error) {
	current, err := sessionActor(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := u.validateCreate(input)
	if err != nil {
		return nil, err
	}
	var workspaceID *string
	if !normalized.AllWorkspaces {
		id, err := u.resolveWorkspaceID(ctx, current, normalized)
		if err != nil {
			return nil, err
		}
		workspaceID = &id
	}

	secret, displayPrefix, hash, err := generateSecret()
	if err != nil {
		return nil, err
	}
	token, err := u.tokens.CreateAccessToken(ctx, ports.CreateAccessTokenInput{
		UserID: current.User.ID, WorkspaceID: workspaceID, Name: normalized.Name, Description: normalized.Description,
		Scope: normalized.Scope, CanCommit: normalized.CanCommit, CanDelete: normalized.CanDelete,
		ManageWorkspaces: normalized.ManageWorkspaces, TokenPrefix: displayPrefix, SecretHash: hash, ExpiresAt: normalized.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return &Created{Token: *token, Secret: secret}, nil
}

func (u *UseCase) resolveWorkspaceID(ctx context.Context, current entities.CurrentActor, input CreateInput) (string, error) {
	workspace, err := u.workspaces.GetWorkspace(ctx, input.WorkspaceIdentity)
	if errors.Is(err, ports.ErrNotFound) {
		return "", domainerrors.NotFound("workspace_not_found", "Workspace not found")
	}
	if err != nil {
		return "", err
	}
	role, err := u.workspaces.WorkspaceRole(ctx, workspace.ID, current.User.ID, current.PlatformAdmin)
	if err != nil {
		return "", err
	}
	if role == "" {
		return "", domainerrors.Forbidden("workspace_forbidden", "Workspace access is forbidden")
	}
	if input.Scope == entities.AccessTokenScopeWrite && !shared.CanWrite(role) {
		return "", domainerrors.Forbidden("access_token_scope_exceeds_role", "Write token requires Workspace Editor role")
	}
	if input.CanCommit && !shared.CanAdmin(role) {
		return "", domainerrors.Forbidden("access_token_owner_admin_required", "Commit permission requires Workspace Admin role")
	}
	return workspace.ID, nil
}

// List возвращает токены текущего пользователя; all=true доступен только Platform Admin.
func (u *UseCase) List(ctx context.Context, all bool) ([]entities.AccessToken, error) {
	current, err := sessionActor(ctx)
	if err != nil {
		return nil, err
	}
	if all {
		if !current.PlatformAdmin {
			return nil, domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required")
		}
		return u.tokens.ListAccessTokens(ctx, nil)
	}
	return u.tokens.ListAccessTokens(ctx, &current.User.ID)
}

// Revoke отзывает свой токен; Platform Admin может отозвать любой. Чужой токен для
// остальных неотличим от отсутствующего, чтобы не раскрывать идентификаторы.
func (u *UseCase) Revoke(ctx context.Context, id string) error {
	current, err := sessionActor(ctx)
	if err != nil {
		return err
	}
	token, err := u.tokens.GetAccessToken(ctx, strings.TrimSpace(id))
	if errors.Is(err, ports.ErrNotFound) {
		return domainerrors.NotFound("access_token_not_found", "Access token not found")
	}
	if err != nil {
		return err
	}
	if token.Owner.ID != current.User.ID && !current.PlatformAdmin {
		return domainerrors.NotFound("access_token_not_found", "Access token not found")
	}
	return u.tokens.RevokeAccessToken(ctx, token.ID, current.User.ID)
}

// Authenticate проверяет секрет из заголовка Authorization. Ошибки — 401 с кодами
// access_token_invalid, access_token_revoked и access_token_expired. Секрет не логируется.
func (u *UseCase) Authenticate(ctx context.Context, secret string) (*Authenticated, error) {
	if !validSecretFormat(secret) {
		return nil, invalidToken()
	}
	hash := hashSecret(secret)
	credential, err := u.tokens.FindAccessTokenCredential(ctx, hash)
	if errors.Is(err, ports.ErrNotFound) {
		return nil, invalidToken()
	}
	if err != nil {
		return nil, err
	}
	// Поиск идёт по уникальному индексу хэша; сравнение за постоянное время — защита
	// на случай, если способ поиска изменится.
	if subtle.ConstantTimeCompare(credential.SecretHash, hash) != 1 {
		return nil, invalidToken()
	}
	now := u.now()
	switch credential.Token.Status(now) {
	case entities.AccessTokenStatusRevoked:
		return nil, domainerrors.Unauthorized("access_token_revoked", "Access token is revoked")
	case entities.AccessTokenStatusExpired:
		return nil, domainerrors.Unauthorized("access_token_expired", "Access token is expired")
	}
	if last := credential.Token.LastUsedAt; last == nil || now.Sub(*last) >= LastUsedInterval {
		// last_used_at — справочная отметка; её сбой не должен отклонять запрос агента.
		_ = u.tokens.TouchAccessToken(ctx, credential.Token.ID, LastUsedInterval)
	}
	token := credential.Token
	return &Authenticated{
		Principal: entities.AccessTokenPrincipal{
			TokenID: token.ID, UserID: credential.Owner.ID, WorkspaceID: token.WorkspaceID, WorkspaceIdentity: token.WorkspaceIdentity, Scope: token.Scope,
			CanCommit: token.CanCommit, CanDelete: token.CanDelete, ExpiresAt: token.ExpiresAt,
			AllWorkspaces: token.AllWorkspaces, ManageWorkspaces: token.ManageWorkspaces,
		},
		Owner: credential.Owner,
	}, nil
}

func (u *UseCase) validateCreate(input CreateInput) (CreateInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Scope = strings.TrimSpace(input.Scope)
	if input.Scope == "" {
		input.Scope = entities.AccessTokenScopeRead
	}
	input.WorkspaceIdentity = strings.TrimSpace(input.WorkspaceIdentity)
	if input.AllWorkspaces && input.WorkspaceIdentity != "" {
		return input, domainerrors.InvalidInput("access_token_workspace_scope_invalid", "Choose either workspaceIdentity or allWorkspaces")
	}
	if !input.AllWorkspaces && input.WorkspaceIdentity == "" {
		return input, domainerrors.InvalidInput("workspace_required", "workspaceIdentity is required for a workspace token")
	}
	if input.ManageWorkspaces && !input.AllWorkspaces {
		return input, domainerrors.InvalidInput("access_token_workspace_scope_invalid", "Workspace management requires allWorkspaces")
	}
	if input.Name == "" || utf8.RuneCountInString(input.Name) > maxNameRunes {
		return input, domainerrors.InvalidInput("access_token_name_invalid", "name must contain 1 to 120 characters")
	}
	if utf8.RuneCountInString(input.Description) > maxDescription {
		return input, domainerrors.InvalidInput("access_token_description_invalid", "description must contain at most 1000 characters")
	}
	if input.Scope != entities.AccessTokenScopeRead && input.Scope != entities.AccessTokenScopeWrite {
		return input, domainerrors.InvalidInput("access_token_scope_invalid", "scope must be read or write")
	}
	if (input.CanCommit || input.CanDelete || input.ManageWorkspaces) && input.Scope != entities.AccessTokenScopeWrite {
		return input, domainerrors.InvalidInput("access_token_flags_require_write", "commit, delete and manageWorkspaces require write scope")
	}
	if input.ExpiresAt.IsZero() {
		return input, domainerrors.InvalidInput("access_token_expiry_required", "expiresAt is required")
	}
	now := u.now()
	if input.ExpiresAt.Before(now.Add(minimumLifetime)) {
		return input, domainerrors.InvalidInput("access_token_expiry_invalid", "expiresAt must be in the future")
	}
	if input.ExpiresAt.After(now.Add(u.maxTTL)) {
		return input, domainerrors.WithDetails(
			domainerrors.InvalidInput("access_token_expiry_too_long", "expiresAt exceeds the maximum token lifetime"),
			map[string]any{"maxTtlSeconds": int64(u.maxTTL / time.Second)},
		)
	}
	input.ExpiresAt = input.ExpiresAt.UTC()
	return input, nil
}

// sessionActor запрещает управление токенами через сам токен: агент не может
// продлить или размножить свой доступ.
func sessionActor(ctx context.Context) (entities.CurrentActor, error) {
	if _, viaToken := entities.AccessTokenFromContext(ctx); viaToken {
		return entities.CurrentActor{}, domainerrors.Forbidden("access_token_management_forbidden", "Access tokens can be managed only from a user session")
	}
	return shared.Actor(ctx)
}

func invalidToken() error {
	return domainerrors.Unauthorized("access_token_invalid", "Access token is invalid")
}
