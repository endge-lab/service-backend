package backend_connections

import (
	"context"
	"net/url"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

// ListResult содержит каталог и право текущего пользователя на его изменение.
type ListResult struct {
	Items     []entities.BackendConnection
	CanManage bool
}

// UseCase управляет глобальным каталогом backend-подключений.
type UseCase struct {
	repository ports.BackendConnectionRepository
}

// NewUseCase создаёт сценарии каталога backend-подключений.
func NewUseCase(repository ports.BackendConnectionRepository) *UseCase {
	return &UseCase{repository: repository}
}

// List возвращает каталог любому аутентифицированному пользователю.
func (s *UseCase) List(ctx context.Context) (ListResult, error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return ListResult{}, err
	}
	items, err := s.repository.ListBackendConnections(ctx)
	return ListResult{Items: items, CanManage: current.PlatformAdmin}, err
}

// Create нормализует и добавляет именованный URL. Проверка доступности намеренно выполняется только браузером.
func (s *UseCase) Create(ctx context.Context, name, baseURL string) (*entities.BackendConnection, error) {
	current, err := platformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	name, err = NormalizeName(name)
	if err != nil {
		return nil, err
	}
	normalized, err := NormalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	value := entities.BackendConnection{ID: uuid.NewString(), Name: name, BaseURL: normalized, CreatedBy: current.User.ID}
	created, err := s.repository.InsertBackendConnection(ctx, value)
	return created, shared.MapConflict(err)
}

// NormalizeName приводит пользовательское название к хранимой форме.
func NormalizeName(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", domainerrors.InvalidInput("backend_name_required", "name is required")
	}
	if len([]rune(value)) > 160 {
		return "", domainerrors.InvalidInput("backend_name_too_long", "name must not exceed 160 characters")
	}
	return value, nil
}

// Delete физически удаляет подключение по id.
func (s *UseCase) Delete(ctx context.Context, id string) error {
	if _, err := platformAdmin(ctx); err != nil {
		return err
	}
	if _, err := uuid.Parse(id); err != nil {
		return domainerrors.NotFound("backend_connection_not_found", "Backend connection not found")
	}
	return shared.MapNotFound(s.repository.DeleteBackendConnection(ctx, id))
}

// NormalizeBaseURL приводит разрешённый абсолютный http/https URL к форме каталога.
func NormalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", domainerrors.InvalidInput("backend_url_invalid", "baseUrl must be an absolute http or https URL")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", domainerrors.InvalidInput("backend_url_scheme_invalid", "baseUrl must use http or https")
	}
	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || strings.Contains(trimmed, "#") {
		return "", domainerrors.InvalidInput("backend_url_components_forbidden", "baseUrl must not contain userinfo, query or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

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
