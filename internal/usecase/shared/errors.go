package shared

import (
	"errors"
	"strings"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

// PreconditionRequired создаёт ошибку отсутствующей обязательной ожидаемой ревизии.
func PreconditionRequired() error {
	return domainerrors.New("precondition_required", "If-Match header is required", 428)
}

// RevisionConflict создаёт ошибку несовпадения ожидаемой и фактической ревизий.
func RevisionConflict() error {
	return domainerrors.Conflict("revision_conflict", "Document revision does not match If-Match")
}

// MapNotFound преобразует ошибку отсутствующей записи в доменную ошибку.
func MapNotFound(err error) error {
	if errors.Is(err, ports.ErrNotFound) {
		return domainerrors.NotFound("not_found", "Entity not found")
	}
	return err
}

// MapConflict преобразует конфликт хранилища в доменную ошибку.
func MapConflict(err error) error {
	if err == nil {
		return nil
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "revision conflict") {
		return RevisionConflict()
	}
	if strings.Contains(text, "duplicate") || strings.Contains(text, "unique constraint") || strings.Contains(text, "23505") {
		return domainerrors.Conflict("identity_conflict", "Identity already exists")
	}
	if strings.Contains(text, "relation target") {
		return domainerrors.InvalidInput("relation_target_not_found", err.Error())
	}
	return err
}
