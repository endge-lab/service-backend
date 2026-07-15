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
func (c *Component) Create(ctx context.Context, input adapters.CreateComponentInput) (result *adapters.ComponentWithFolder, err error) {

	const op = "component.create"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateCreateInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	folder, err := c.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.FolderIdentity),
		)
		return nil, err
	}

	exists, err := c.componentRepository.ExistsByIdentity(
		ctx,
		project.ID,
		input.Identity,
	)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.Identity),
		)
		return nil, err
	}
	if exists {
		err = apperrors.Conflict(
			"identity_conflict",
			"component identity already exists",
		)
		logOperationError(observed.Logger(), op, err,
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

	componentResult, err := c.componentRepository.Create(ctx, component)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.Identity),
		)
		return nil, err
	}
	observed.Logger().Debug("component created",
		zap.String("component_id", componentResult.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("component_identity", componentResult.Identity),
		zap.String("folder_id", folder.ID.String()),
		zap.String("folder_identity", folder.Identity),
		zap.String("component_type", string(componentResult.ComponentType)),
	)

	return componentWithFolder(componentResult, folder.Identity), nil
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
func (c *Component) Update(ctx context.Context, input adapters.UpdateComponentInput) (result *adapters.ComponentWithFolder, err error) {

	const op = "component.update"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateUpdateInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	current, err := c.componentRepository.GetByIdentity(ctx, project.ID, input.ComponentIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentIdentity))
		return nil, err
	}

	folder, err := c.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.FolderIdentity),
		)

		return nil, err
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

	componentResult, err := c.componentRepository.Update(ctx, updated)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", current.Identity),
		)
		return nil, err
	}
	observed.Logger().Debug("component updated",
		zap.String("component_id", componentResult.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("component_identity", componentResult.Identity),
		zap.String("folder_id", folder.ID.String()),
		zap.String("folder_identity", folder.Identity),
		zap.String("component_type", string(componentResult.ComponentType)),
	)

	return componentWithFolder(componentResult, folder.Identity), nil

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
) (result *adapters.ComponentWithFolder, err error) {
	const op = "component.get_by_identity"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateIdentityInput(
		&input.ProjectIdentity,
		&input.ComponentIdentity,
	); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}

	project, err := c.projectRepository.GetByIdentity(
		ctx,
		input.ProjectIdentity,
	)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	component, err := c.componentRepository.GetByIdentity(
		ctx,
		project.ID,
		input.ComponentIdentity,
	)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentIdentity),
		)
		return nil, err
	}

	folder, err := c.folderRepository.GetByID(ctx, component.FolderID)
	if err != nil {
		relationErr := apperrors.Internal(
			"component_folder_not_found",
			"component references an unavailable folder",
		)
		logOperationError(observed.Logger(), op, relationErr,
			zap.NamedError("repository_error", err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_id", component.ID.String()),
			zap.String("component_identity", component.Identity),
			zap.String("folder_id", component.FolderID.String()),
		)
		return nil, relationErr
	}
	observed.Logger().Debug("component retrieved",
		zap.String("component_id", component.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("component_identity", component.Identity),
		zap.String("folder_id", component.FolderID.String()),
		zap.String("folder_identity", folder.Identity),
	)

	return componentWithFolder(component, folder.Identity), nil
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
func (c *Component) List(ctx context.Context, input adapters.ListComponentsInput) (result []*adapters.ComponentWithFolder, err error) {
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
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}

	filter := ports.ComponentsFilter{
		ProjectID:     project.ID,
		ComponentType: input.ComponentType,
	}

	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		)
		return nil, err
	}

	components, err := c.componentRepository.List(ctx, filter)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	if len(components) == 0 {
		observed.Logger().Debug("components listed",
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
			zap.Int("count", 0),
		)
		return make([]*adapters.ComponentWithFolder, 0), nil
	}

	folders, err := c.folderRepository.List(ctx, &project.ID, entities.FolderEntityTypeComponents)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}

	result, err = componentWithFolders(components, folders)
	if err != nil {
		fields := []zap.Field{
			zap.String("project_identity", input.ProjectIdentity),
			zap.Int("component_count", len(components)),
		}
		if component := firstComponentWithUnavailableFolder(components, folders); component != nil {
			fields = append(fields,
				zap.String("component_id", component.ID.String()),
				zap.String("component_identity", component.Identity),
				zap.String("folder_id", component.FolderID.String()),
			)
		}
		logOperationError(observed.Logger(), op, err, fields...)
		return nil, err
	}
	observed.Logger().Debug("components listed",
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		zap.Int("count", len(result)),
	)

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
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentIdentity),
		)
		return err
	}
	if err = c.componentRepository.SoftDelete(ctx, component.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_id", component.ID.String()),
			zap.String("component_identity", component.Identity),
		)
		return err
	}
	observed.Logger().Debug("component soft deleted",
		zap.String("component_id", component.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("component_identity", component.Identity),
	)
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
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentIdentity),
		)
		return err
	}
	if err = c.componentRepository.Restore(ctx, component.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_id", component.ID.String()),
			zap.String("component_identity", component.Identity),
		)
		return err
	}
	observed.Logger().Debug("component restored",
		zap.String("component_id", component.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("component_identity", component.Identity),
	)
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
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentIdentity),
		)
		return err
	}
	if err = c.componentRepository.HardDelete(ctx, component.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_id", component.ID.String()),
			zap.String("component_identity", component.Identity),
		)
		return err
	}
	observed.Logger().Debug("component hard deleted",
		zap.String("component_id", component.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("component_identity", component.Identity),
	)
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
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}
	filter := ports.ComponentsFilter{ProjectID: project.ID, ComponentType: input.ComponentType}
	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		)
		return 0, err
	}
	count, err = c.componentRepository.Count(ctx, filter)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}
	observed.Logger().Debug("components counted",
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		zap.Int64("count", count),
	)
	return count, nil
}
