// Package shared содержит только общие application-layer политики контекста.
// Бизнес-сценарии и транзакции остаются в пакетах-владельцах use case.
package shared

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// Actor возвращает текущего аутентифицированного пользователя из контекста.
func Actor(ctx context.Context) (entities.CurrentActor, error) {
	value, ok := entities.CurrentActorFromContext(ctx)
	if !ok || value.User == nil {
		return value, domainerrors.Unauthorized("auth.current_user_missing", "Current user is missing")
	}
	return value, nil
}

// Access возвращает рабочее пространство и роль текущего пользователя.
func Access(ctx context.Context) (entities.WorkspaceAccess, error) {
	value, ok := entities.WorkspaceAccessFromContext(ctx)
	if !ok {
		return value, domainerrors.InvalidInput("workspace_required", "Workspace context is required")
	}
	return value, nil
}

// WriteContext возвращает контекст записи и проверяет право изменять рабочее пространство.
func WriteContext(ctx context.Context) (entities.CurrentActor, entities.WorkspaceAccess, error) {
	current, err := Actor(ctx)
	if err != nil {
		return current, entities.WorkspaceAccess{}, err
	}
	scope, err := Access(ctx)
	if err != nil {
		return current, scope, err
	}
	if token, viaToken := entities.AccessTokenFromContext(ctx); viaToken && !token.CanWrite() {
		return current, scope, domainerrors.Forbidden("access_token_scope_insufficient", "Access token scope does not allow changes")
	}
	if !CanWrite(scope.Role) {
		return current, scope, domainerrors.Forbidden("workspace_editor_required", "Workspace Editor role is required")
	}
	return current, scope, nil
}

// AdminContext возвращает контекст административной операции рабочего пространства.
func AdminContext(ctx context.Context) (entities.CurrentActor, entities.WorkspaceAccess, error) {
	current, err := Actor(ctx)
	if err != nil {
		return current, entities.WorkspaceAccess{}, err
	}
	scope, err := Access(ctx)
	if err != nil {
		return current, scope, err
	}
	if _, viaToken := entities.AccessTokenFromContext(ctx); viaToken {
		return current, scope, domainerrors.Forbidden("access_token_admin_forbidden", "Administrative operations are not available to access tokens")
	}
	if !CanAdmin(scope.Role) {
		return current, scope, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	return current, scope, nil
}

// CommitContext возвращает контекст создания коммита.
// Сессия пользователя требует роль Admin (как AdminContext). Персональный токен
// коммитит только со scope write и флагом commit и только если его владелец —
// Admin этого workspace или Platform Admin: права токена не выше прав человека.
func CommitContext(ctx context.Context) (entities.CurrentActor, entities.WorkspaceAccess, error) {
	token, viaToken := entities.AccessTokenFromContext(ctx)
	if !viaToken {
		return AdminContext(ctx)
	}
	current, scope, err := WriteContext(ctx)
	if err != nil {
		return current, scope, err
	}
	if !token.CanCommit {
		return current, scope, domainerrors.Forbidden("access_token_commit_forbidden", "Access token does not allow commits and releases")
	}
	if !CanAdmin(scope.OwnerRole) {
		return current, scope, domainerrors.Forbidden("access_token_owner_admin_required", "Access token owner must be Workspace Admin to commit")
	}
	return current, scope, nil
}

// CanWrite проверяет, разрешает ли роль изменять рабочее пространство.
func CanWrite(role string) bool {
	return role == "editor" || role == "admin" || role == "platform_admin"
}

// CanAdmin проверяет, обладает ли роль административными правами.
func CanAdmin(role string) bool { return role == "admin" || role == "platform_admin" }
