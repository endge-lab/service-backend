package integrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

// Create создаёт интеграцию и записывает ревизию.
func (s *UseCase) Create(ctx context.Context, input CreateInput) (result *entities.Integration, err error) {
	current, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	values, err := inputValues(input)
	if err != nil {
		return nil, err
	}
	if err = shared.RejectReadOnly(values); err != nil {
		return nil, err
	}
	if err = shared.ValidateSecrets(values); err != nil {
		return nil, err
	}
	if err = validate(values); err != nil {
		return nil, err
	}
	value := entities.Integration{ID: uuid.NewString(), Identity: text(values, "identity"), DisplayName: text(values, "displayName"), Description: optional(values, "description"), Version: text(values, "version"), ManagedBy: fallback(text(values, "managedBy"), "user"), ManagedByID: optional(values, "managedById"), Meta: raw(values["meta"], `{}`), Active: boolean(values, "active", true), Revision: 1, CreatedBy: entities.Actor{ID: current.User.ID}, UpdatedBy: entities.Actor{ID: current.User.ID}}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		created, txErr := s.repository.InsertIntegration(txctx, value)
		if txErr != nil {
			return txErr
		}
		if txErr = s.history.RecordIntegration(txctx, *created, "create", nil); txErr != nil {
			return txErr
		}
		result = created
		return nil
	})
	return result, shared.MapConflict(err)
}

// Patch частично обновляет интеграцию с проверкой ожидаемой ревизии.
func (s *UseCase) Patch(ctx context.Context, identity string, input PatchInput, expected int) (result *entities.Integration, err error) {
	current, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	patch, err := input.values()
	if err != nil {
		return nil, err
	}
	if err = shared.RejectReadOnly(patch); err != nil {
		return nil, err
	}
	if err = shared.ValidateSecrets(patch); err != nil {
		return nil, err
	}
	existing, err := s.repository.GetIntegration(ctx, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	next := *existing
	if value, ok := patch["identity"].(string); ok {
		next.Identity = strings.TrimSpace(value)
	}
	if value, ok := patch["displayName"].(string); ok {
		next.DisplayName = value
	}
	if _, ok := patch["description"]; ok {
		next.Description = optional(patch, "description")
	}
	if value, ok := patch["version"].(string); ok {
		next.Version = strings.TrimSpace(value)
	}
	if value, ok := patch["managedBy"].(string); ok {
		next.ManagedBy = value
	}
	if _, ok := patch["managedById"]; ok {
		next.ManagedByID = optional(patch, "managedById")
	}
	if value, ok := patch["meta"]; ok {
		next.Meta = raw(value, `{}`)
	}
	if value, ok := patch["active"].(bool); ok {
		next.Active = value
	}
	next.UpdatedBy = entities.Actor{ID: current.User.ID}
	if err = validate(map[string]any{"identity": next.Identity, "displayName": next.DisplayName, "version": next.Version, "managedBy": next.ManagedBy}); err != nil {
		return nil, err
	}
	if digest(*existing) == digest(next) {
		return existing, nil
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateIntegration(txctx, next, expected)
		if txErr != nil {
			return txErr
		}
		if txErr = s.history.RecordIntegration(txctx, *updated, "update", nil); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

// Delete мягко удаляет интеграцию с проверкой ожидаемой ревизии.
func (s *UseCase) Delete(ctx context.Context, identity string, expected int) (*entities.Integration, error) {
	return s.setDeleted(ctx, identity, expected, true)
}

// Restore восстанавливает мягко удалённую интеграцию.
func (s *UseCase) Restore(ctx context.Context, identity string, expected int) (*entities.Integration, error) {
	return s.setDeleted(ctx, identity, expected, false)
}

// setDeleted устанавливает или снимает отметку мягкого удаления.
func (s *UseCase) setDeleted(ctx context.Context, identity string, expected int, deleted bool) (result *entities.Integration, err error) {
	current, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	existing, err := s.repository.GetIntegration(ctx, identity, true)
	if err != nil {
		return nil, shared.MapNotFound(err)
	}
	if existing.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	if (existing.DeletedAt != nil) == deleted {
		return existing, nil
	}
	next := *existing
	next.UpdatedBy = entities.Actor{ID: current.User.ID}
	operation := "restore"
	if deleted {
		now := time.Now().UTC()
		next.DeletedAt, next.Active, operation = &now, false, "delete"
	} else {
		next.DeletedAt, next.Active = nil, true
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.repository.UpdateIntegration(txctx, next, expected)
		if txErr != nil {
			return txErr
		}
		if txErr = s.history.RecordIntegration(txctx, *updated, operation, nil); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

// platformAdmin проверяет права администратора платформы.
func platformAdmin(ctx context.Context) (entities.CurrentActor, error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return current, err
	}
	if !current.PlatformAdmin {
		return current, domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required")
	}
	return current, nil
}

// validate проверяет входные данные интеграции.
func validate(values map[string]any) error {
	identity := text(values, "identity")
	if identity == "" {
		return domainerrors.InvalidInput("identity_required", "identity is required")
	}
	if len(identity) > 160 {
		return domainerrors.InvalidInput("identity_too_long", "identity must not exceed 160 characters")
	}
	if text(values, "displayName") == "" || text(values, "version") == "" {
		return domainerrors.InvalidInput("integration_fields_required", "displayName and version are required")
	}
	managedBy := fallback(text(values, "managedBy"), "user")
	if managedBy != "user" && managedBy != "system" && managedBy != "integration" {
		return domainerrors.InvalidInput("managed_by_invalid", "managedBy is invalid")
	}
	return nil
}

// text извлекает и нормализует текстовое поле интеграции.
func text(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

// optional извлекает необязательное текстовое поле интеграции.
func optional(values map[string]any, key string) *string {
	value, ok := values[key].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return nil
	}
	return &value
}

// fallback возвращает строку или резервное значение.
func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// boolean извлекает логическое поле интеграции.
func boolean(values map[string]any, key string, defaultValue bool) bool {
	value, ok := values[key].(bool)
	if !ok {
		return defaultValue
	}
	return value
}

// raw извлекает поле как JSON.
func raw(value any, defaultValue string) json.RawMessage {
	if value == nil {
		return json.RawMessage(defaultValue)
	}
	result, _ := json.Marshal(value)
	return result
}

// digest вычисляет контрольную сумму интеграции.
func digest(value entities.Integration) string {
	bytes, _ := json.Marshal(map[string]any{"identity": value.Identity, "displayName": value.DisplayName, "description": value.Description, "version": value.Version, "managedBy": value.ManagedBy, "managedById": value.ManagedByID, "meta": value.Meta, "active": value.Active, "deletedAt": value.DeletedAt})
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}
