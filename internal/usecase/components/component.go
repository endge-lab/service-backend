package components

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/ports"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const componentOperationTimeout = 15 * time.Second

type Component struct {
	componentRepository ports.ComponentsRepository
	folderRepository    ports.FoldersRepository
	projectRepository   ports.ProjectsRepository
	observed            shared.ObservedUseCase
}

type ComponentParams struct {
	ComponentRepository ports.ComponentsRepository
	FolderRepository    ports.FoldersRepository
	ProjectRepository   ports.ProjectsRepository
	Tracer              trace.Tracer
	Logger              *zap.Logger
	Metrics             *shared.UseCaseMetrics
}

func NewComponentService(params ComponentParams) *Component {
	return &Component{
		componentRepository: params.ComponentRepository,
		folderRepository:    params.FolderRepository,
		projectRepository:   params.ProjectRepository,
		observed:            shared.NewObservedUseCase(params.Tracer, params.Logger, params.Metrics),
	}
}

// Create создает компонент в указанной папке проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует input, разрешает project и folder components, проверяет identity и создает компонент.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Component) Create(ctx context.Context, input adapters.CreateComponentInput) (result *entities.Component, err error) {

	const op = "components.create"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateCreateInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	folder, err := c.folderRepository.GetByIdentity(
		ctx,
		&project.ID,
		entities.FolderEntityTypeComponents,
		input.FolderIdentity,
	)
	if err != nil {
		observed.Logger().Warn(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.FolderIdentity),
		)
		return nil, apperrors.InvalidInput(
			"folder_entity_type_mismatch",
			"folder must belong to the project and have components entity type",
		)
	}

	exists, err := c.componentRepository.ExistsByIdentity(
		ctx,
		project.ID,
		input.Identity,
	)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("component_identity", input.Identity),
		)
		return nil, err
	}
	if exists {
		err = apperrors.Conflict(
			"identity_conflict",
			"component identity already exists",
		)
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.Identity))
		return nil, err
	}

	component := &entities.Component{
		ProjectID:     project.ID,
		FolderID:      folder.ID,
		Identity:      input.Identity,
		DisplayName:   input.DisplayName,
		Description:   input.Description,
		ComponentType: input.ComponentType,
		Source:        input.Source,
		SourceFormat:  entities.ComponentSourceFormatSFC,
		PropsSchema:   input.PropsSchema,
		Bindings:      input.Bindings,
		Meta:          input.Meta,
		Active:        input.Active,
	}

	result, err = c.componentRepository.Create(ctx, component)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("component_identity", input.Identity),
		)
		return nil, err
	}

	return result, nil
}

// Update обновляет компонент в указанной папке проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует input, получает project, существующий компонент и folder components, затем обновляет компонент.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Component) Update(ctx context.Context, input adapters.UpdateComponentInput) (result *entities.Component, err error) {

	const op = "components.update"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateUpdateInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	current, err := c.componentRepository.GetByIdentity(ctx, project.ID, input.ComponentIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentIdentity))
		return nil, err
	}

	folder, err := c.folderRepository.GetByIdentity(
		ctx,
		&project.ID,
		entities.FolderEntityTypeComponents,
		input.FolderIdentity,
	)
	if err != nil {
		observed.Logger().Warn(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.FolderIdentity),
		)

		return nil, apperrors.InvalidInput(
			"folder_entity_type_mismatch",
			"folder must belong to the project and have components entity type",
		)
	}

	updated := &entities.Component{
		ID:            current.ID,
		ProjectID:     current.ProjectID,
		FolderID:      folder.ID,
		Identity:      current.Identity,
		DisplayName:   input.DisplayName,
		Description:   input.Description,
		ComponentType: input.ComponentType,
		Source:        input.Source,
		SourceFormat:  entities.ComponentSourceFormatSFC,
		PropsSchema:   input.PropsSchema,
		Bindings:      input.Bindings,
		Meta:          input.Meta,
		Active:        input.Active,
		DeletedAt:     current.DeletedAt,
		CreatedAt:     current.CreatedAt,
		UpdatedAt:     current.UpdatedAt,
	}

	result, err = c.componentRepository.Update(ctx, updated)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("component_identity", current.Identity),
		)
		return nil, err
	}

	return result, nil

}

// GetByIdentity возвращает активный компонент по identity в пределах проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует identities, получает project и запрашивает активный компонент.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Component) GetByIdentity(
	ctx context.Context,
	input adapters.GetComponentInput,
) (result *entities.Component, err error) {
	const op = "component.get_by_identity"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateIdentityInput(
		&input.ProjectIdentity,
		&input.ComponentIdentity,
	); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}

	project, err := c.projectRepository.GetByIdentity(
		ctx,
		input.ProjectIdentity,
	)
	if err != nil {
		observed.Logger().Error(
			op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	result, err = c.componentRepository.GetByIdentity(
		ctx,
		project.ID,
		input.ComponentIdentity,
	)
	if err != nil {
		observed.Logger().Error(
			op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentIdentity),
		)
		return nil, err
	}

	return result, nil
}

// List возвращает активные компоненты проекта с учетом фильтров.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует фильтр, получает project и optional folder components, затем запрашивает список.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Component) List(ctx context.Context, input adapters.ListComponentsInput) (result []*entities.Component, err error) {
	const op = "component.list"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(
		&input.ProjectIdentity,
		input.FolderIdentity,
		input.ComponentType,
	); err != nil {
		return nil, err
	}

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err))
		return nil, err
	}

	filter := ports.ComponentsFilter{
		ProjectID:     project.ID,
		ComponentType: input.ComponentType,
	}

	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		return nil, err
	}

	result, err = c.componentRepository.List(ctx, filter)
	if err != nil {
		observed.Logger().Error(op,
			zap.Error(err),
			zap.String("project_identity", input.ProjectIdentity))
	}

	return result, err
}

// SoftDelete выполняет мягкое удаление компонента.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Разрешает component identity в пределах проекта и выполняет мягкое удаление.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Component) SoftDelete(ctx context.Context, input adapters.ComponentIdentityInput) (err error) {
	const op = "component.soft_delete"
	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	component, err := c.resolveComponent(ctx, input, false)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return err
	}
	if err = c.componentRepository.SoftDelete(ctx, component.ID); err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("component_identity", component.Identity))
		return err
	}
	return nil
}

// Restore восстанавливает мягко удаленный компонент.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Разрешает component identity с учетом soft-delete и восстанавливает компонент.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Component) Restore(ctx context.Context, input adapters.ComponentIdentityInput) (err error) {
	const op = "component.restore"
	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	component, err := c.resolveComponent(ctx, input, true)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return err
	}
	if err = c.componentRepository.Restore(ctx, component.ID); err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("component_identity", component.Identity))
		return err
	}
	return nil
}

// HardDelete физически удаляет компонент.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Разрешает component identity с учетом soft-delete и физически удаляет компонент.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Component) HardDelete(ctx context.Context, input adapters.ComponentIdentityInput) (err error) {
	const op = "component.hard_delete"
	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	component, err := c.resolveComponent(ctx, input, true)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return err
	}
	if err = c.componentRepository.HardDelete(ctx, component.ID); err != nil {
		observed.Logger().Error(op, zap.Error(err), zap.String("component_identity", component.Identity))
		return err
	}
	return nil
}

// Count возвращает количество активных компонентов с учетом фильтров.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует фильтр, разрешает project и optional folder components, затем подсчитывает компоненты.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Component) Count(ctx context.Context, input adapters.ListComponentsInput) (count int64, err error) {
	const op = "component.count"
	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(&input.ProjectIdentity, input.FolderIdentity, input.ComponentType); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return 0, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return 0, err
	}
	filter := ports.ComponentsFilter{ProjectID: project.ID, ComponentType: input.ComponentType}
	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		return 0, err
	}
	count, err = c.componentRepository.Count(ctx, filter)
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return 0, err
	}
	return count, nil
}
