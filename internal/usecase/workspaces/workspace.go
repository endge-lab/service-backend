package workspaces

import (
	"context"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	apperrors "github.com/endge-lab/service-kit-go/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const operationTimeout = 15 * time.Second

type Workspace struct {
	repository ports.WorkspacesRepository
	observed   shared.ObservedUseCase
}
type WorkspaceParams struct {
	Repository ports.WorkspacesRepository
	Tracer     trace.Tracer
	Logger     *zap.Logger
	Metrics    *shared.UseCaseMetrics
}

func NewWorkspaceService(params WorkspaceParams) *Workspace {
	return &Workspace{repository: params.Repository, observed: shared.NewObservedUseCase(params.Tracer, params.Logger, params.Metrics)}
}

// Create валидирует вход и создаёт корневой workspace.
//
// Параметры:
//
//	ctx - контекст выполнения операции;
//	input - identity, displayName и опциональная полная configuration нового
//	workspace.
//
// Что делает функция:
//
//	Нормализует identity и displayName, проверяет их непустоту. Если
//	configuration отсутствует, подставляет system default с locales ru/en,
//	themes light/dark и adapter native-vue. Если configuration передана,
//	использует её как полную root configuration, без patch или merge.
//	Валидирует arrays, уникальность vars/locales/themes/adapters, ссылки на
//	default locale/theme/adapter и правила optional SSE. После успешной
//	валидации создаёт RWorkspace и передаёт его в WorkspacesRepository.
//
// Возвращаемые значения:
//
//	*entities.RWorkspace - созданный workspace с system-generated ID и
//	timestamps;
//	error - validation/conflict ошибка либо ошибка repository.
func (s *Workspace) Create(ctx context.Context, input CreateWorkspaceInput) (result *entities.RWorkspace, err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	ctx, observed := s.observed.StartObservedOperation(ctx, "workspace.create", nil, nil)
	defer observed.End(&err)
	if err = normalizeCreateInput(&input); err != nil {
		observed.Logger().Warn("workspace create validation failed", zap.Error(err))
		return nil, err
	}
	configuration := entities.DefaultEndgeConfiguration()
	if input.Configuration != nil {
		configuration = *input.Configuration
	}
	if err = validateConfiguration(configuration); err != nil {
		observed.Logger().Warn("workspace configuration validation failed", zap.Error(err))
		return nil, err
	}
	result, err = s.repository.Create(ctx, &entities.RWorkspace{Identity: input.Identity, DisplayName: input.DisplayName, Configuration: configuration})
	if err != nil {
		observed.Logger().Error("workspace create failed", zap.Error(err))
		return nil, err
	}
	observed.Logger().Debug("workspace created", zap.String("workspace_id", result.ID.String()), zap.String("identity", result.Identity))
	return result, nil
}

// List возвращает все workspace без фильтрации по пользователю.
//
// Параметры:
//
//	ctx - контекст выполнения операции.
//
// Что делает функция:
//
//	Не разрешает user, membership или role: эти механизмы не входят в задачу
//	04. Делегирует получение списка WorkspacesRepository и возвращает полные
//	configuration каждого workspace без вычисления effective configuration.
//
// Возвращаемые значения:
//
//	[]*entities.RWorkspace - все созданные workspace;
//	error - ошибка repository.
func (s *Workspace) List(ctx context.Context) (result []*entities.RWorkspace, err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	ctx, observed := s.observed.StartObservedOperation(ctx, "workspace.list", nil, nil)
	defer observed.End(&err)
	result, err = s.repository.List(ctx)
	if err != nil {
		observed.Logger().Error("workspace list failed", zap.Error(err))
		return nil, err
	}
	observed.Logger().Debug("workspaces listed", zap.Int("count", len(result)))
	return result, nil
}

// GetByIdentity возвращает workspace по стабильному identity.
//
// Параметры:
//
//	ctx - контекст выполнения операции;
//	identity - пользовательский identity workspace из path/request context.
//
// Что делает функция:
//
//	Удаляет пробелы вокруг identity, проверяет непустое значение и запрашивает
//	workspace через WorkspacesRepository. Не проверяет access control и не
//	собирает configuration cascade: workspace.configuration является полной
//	root configuration и возвращается как сохранена.
//
// Возвращаемые значения:
//
//	*entities.RWorkspace - найденный workspace;
//	error - validation_error, not_found либо ошибка repository.
func (s *Workspace) GetByIdentity(ctx context.Context, identity string) (result *entities.RWorkspace, err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	identity = strings.TrimSpace(identity)
	ctx, observed := s.observed.StartObservedOperation(ctx, "workspace.get_by_identity", nil, nil)
	defer observed.End(&err)
	if identity == "" {
		return nil, apperrors.InvalidInput("validation_error", "workspace identity is required")
	}
	result, err = s.repository.GetByIdentity(ctx, identity)
	if err != nil {
		observed.Logger().Error("workspace get failed", zap.Error(err))
		return nil, err
	}
	return result, nil
}

// Update частично обновляет верхнеуровневые поля workspace.
//
// Параметры:
//
//	ctx - контекст выполнения операции;
//	input - identity workspace и опциональные displayName и полная
//	configuration.
//
// Что делает функция:
//
//	Нормализует identity, проверяет, что передан хотя бы один изменяемый
//	параметр, и разрешает текущий workspace через WorkspacesRepository.
//	Identity, ID и createdAt из текущей записи сохраняются неизменными. При
//	передаче displayName заменяет только displayName. При передаче
//	configuration валидирует весь документ и полностью заменяет сохранённую
//	root configuration; частичный JSON merge намеренно не выполняется. Затем
//	передаёт итоговую сущность в repository.
//
// Возвращаемые значения:
//
//	*entities.RWorkspace - обновлённый workspace;
//	error - validation_error, not_found либо ошибка repository.
func (s *Workspace) Update(ctx context.Context, input UpdateWorkspaceInput) (result *entities.RWorkspace, err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	ctx, observed := s.observed.StartObservedOperation(ctx, "workspace.update", nil, nil)
	defer observed.End(&err)
	if err = normalizeUpdateInput(&input); err != nil {
		observed.Logger().Warn("workspace update validation failed", zap.Error(err))
		return nil, err
	}
	current, err := s.repository.GetByIdentity(ctx, input.Identity)
	if err != nil {
		observed.Logger().Error("workspace resolve failed", zap.Error(err))
		return nil, err
	}
	updated := *current
	if input.DisplayName != nil {
		updated.DisplayName = *input.DisplayName
	}
	if input.Configuration != nil {
		if err = validateConfiguration(*input.Configuration); err != nil {
			observed.Logger().Warn("workspace configuration validation failed", zap.Error(err))
			return nil, err
		}
		updated.Configuration = *input.Configuration
	}
	result, err = s.repository.Update(ctx, &updated)
	if err != nil {
		observed.Logger().Error("workspace update failed", zap.Error(err))
		return nil, err
	}
	observed.Logger().Debug("workspace updated", zap.String("workspace_id", result.ID.String()))
	return result, nil
}
