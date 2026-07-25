package components_legacy

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"go.uber.org/zap"
)

const componentOperationTimeout = 15 * time.Second

type ComponentLegacy struct {
	componentRepository ports.ComponentsLegacyRepository
	folderRepository    ports.FoldersRepository
	projectRepository   ports.ProjectsRepository
	observer            observability.Observer
}

type ComponentLegacyParams struct {
	ComponentLegacyRepository ports.ComponentsLegacyRepository
	FolderRepository          ports.FoldersRepository
	ProjectRepository         ports.ProjectsRepository
	Observability             *observability.Core
	Metrics                   *shared.UseCaseMetrics
}

func NewComponentLegacyService(params ComponentLegacyParams) *ComponentLegacy {
	return &ComponentLegacy{
		componentRepository: params.ComponentLegacyRepository,
		folderRepository:    params.FolderRepository,
		projectRepository:   params.ProjectRepository,
		observer:            params.Observability.For(observability.LayerUseCase, "components_legacy_usecase").WithRecorder(params.Metrics),
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
func (c *ComponentLegacy) Create(ctx context.Context, input CreateComponentLegacyInput) (result *ComponentLegacyWithFolder, err error) {

	const op = "component.create"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateCreateInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "component create input validated", nil, zap.String("project_identity", input.ProjectIdentity), zap.String("component_identity", input.Identity))

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for component create", nil, zap.String("project_id", project.ID.String()))

	folder, err := c.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.FolderIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".folder_resolved", "folder resolved for component create", nil, zap.String("folder_id", folder.ID.String()), zap.String("folder_identity", folder.Identity))

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
	observed.RecordStep(op+".identity_available", "component identity availability confirmed", nil, zap.String("component_identity", input.Identity))

	component := &entities.RComponentLegacy{
		WorkspaceID:   project.WorkspaceID,
		ProjectID:     project.ID,
		FolderID:      folder.ID,
		Identity:      input.Identity,
		DisplayName:   input.DisplayName,
		Description:   input.Description,
		ComponentType: input.ComponentType,
		Source:        input.Source,
		SourceFormat:  entities.RComponentLegacySourceFormatSFC,
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
	observed.RecordStep(op+".persisted", "component created", nil,
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
func (c *ComponentLegacy) Update(ctx context.Context, input UpdateComponentLegacyInput) (result *ComponentLegacyWithFolder, err error) {

	const op = "component.update"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateUpdateInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "component update input validated", nil, zap.String("project_identity", input.ProjectIdentity), zap.String("component_identity", input.ComponentLegacyIdentity))

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for component update", nil, zap.String("project_id", project.ID.String()))

	current, err := c.componentRepository.GetByIdentity(ctx, project.ID, input.ComponentLegacyIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentLegacyIdentity))
		return nil, err
	}
	observed.RecordStep(op+".current_resolved", "component resolved for update", nil, zap.String("component_id", current.ID.String()), zap.String("component_identity", current.Identity))

	folder, err := c.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", input.FolderIdentity),
		)

		return nil, err
	}
	observed.RecordStep(op+".folder_resolved", "folder resolved for component update", nil, zap.String("folder_id", folder.ID.String()), zap.String("folder_identity", folder.Identity))

	updated := &entities.RComponentLegacy{
		ID:            current.ID,
		WorkspaceID:   current.WorkspaceID,
		ProjectID:     current.ProjectID,
		FolderID:      folder.ID,
		Identity:      current.Identity,
		DisplayName:   input.DisplayName,
		Description:   input.Description,
		ComponentType: input.ComponentType,
		Source:        input.Source,
		SourceFormat:  entities.RComponentLegacySourceFormatSFC,
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
	observed.RecordStep(op+".persisted", "component updated", nil,
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
func (c *ComponentLegacy) GetByIdentity(
	ctx context.Context,
	input GetComponentLegacyInput,
) (result *ComponentLegacyWithFolder, err error) {
	const op = "component.get_by_identity"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateIdentityInput(
		&input.ProjectIdentity,
		&input.ComponentLegacyIdentity,
	); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(op+".input_validated", "component identity input validated", nil,
		zap.String("project_identity", input.ProjectIdentity), zap.String("component_identity", input.ComponentLegacyIdentity))

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
	observed.RecordStep(op+".project_resolved", "project resolved for component retrieval", nil,
		zap.String("project_id", project.ID.String()), zap.String("project_identity", project.Identity))

	component, err := c.componentRepository.GetByIdentity(
		ctx,
		project.ID,
		input.ComponentLegacyIdentity,
	)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentLegacyIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".current_resolved", "component resolved for retrieval", nil,
		zap.String("component_id", component.ID.String()), zap.String("component_identity", component.Identity))

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
	observed.RecordStep(op+".folder_resolved", "component folder resolved for retrieval", nil,
		zap.String("folder_id", folder.ID.String()), zap.String("folder_identity", folder.Identity))
	observed.RecordStep(op+".result_loaded", "component retrieved", nil,
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
func (c *ComponentLegacy) List(ctx context.Context, input ListComponentsLegacyInput) (result []*ComponentLegacyWithFolder, err error) {
	const op = "component.list"

	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()

	ctx, observed := c.observer.Start(ctx, op, nil, nil)
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
	observed.RecordStep(op+".input_validated", "component list input validated", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		zap.String("component_type", dereferenceComponentType(input.ComponentType)))

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for component list", nil,
		zap.String("project_id", project.ID.String()), zap.String("project_identity", project.Identity))

	filter := ports.ComponentsLegacyFilter{
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
	observed.RecordStep(op+".folder_filter_resolved", "component folder filter resolved", nil,
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)))

	components, err := c.componentRepository.List(ctx, filter)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	if len(components) == 0 {
		observed.RecordStep(op+".result_loaded", "components listed", nil,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
			zap.Int("count", 0),
		)
		return make([]*ComponentLegacyWithFolder, 0), nil
	}

	folders, err := c.folderRepository.List(ctx, &project.ID, entities.FolderEntityTypeComponentsLegacy)
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
		if component := firstComponentLegacyWithUnavailableFolder(components, folders); component != nil {
			fields = append(fields,
				zap.String("component_id", component.ID.String()),
				zap.String("component_identity", component.Identity),
				zap.String("folder_id", component.FolderID.String()),
			)
		}
		logOperationError(observed.Logger(), op, err, fields...)
		return nil, err
	}
	observed.RecordStep(op+".result_loaded", "components listed", nil,
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
func (c *ComponentLegacy) SoftDelete(ctx context.Context, input ComponentLegacyIdentityInput) (err error) {
	const op = "component.soft_delete"
	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()
	ctx, observed := c.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	observed.RecordStep(op+".input_received", "component state change input received", nil,
		zap.String("project_identity", input.ProjectIdentity), zap.String("component_identity", input.ComponentLegacyIdentity))

	component, err := c.resolveComponentLegacy(ctx, input, false)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentLegacyIdentity),
		)
		return err
	}
	observed.RecordStep(op+".current_resolved", "component resolved for soft delete", nil,
		zap.String("component_id", component.ID.String()), zap.String("component_identity", component.Identity))
	if err = c.componentRepository.SoftDelete(ctx, component.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_id", component.ID.String()),
			zap.String("component_identity", component.Identity),
		)
		return err
	}
	observed.RecordStep(op+".persisted", "component soft deleted", nil,
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
func (c *ComponentLegacy) Restore(ctx context.Context, input ComponentLegacyIdentityInput) (err error) {
	const op = "component.restore"
	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()
	ctx, observed := c.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	observed.RecordStep(op+".input_received", "component state change input received", nil,
		zap.String("project_identity", input.ProjectIdentity), zap.String("component_identity", input.ComponentLegacyIdentity))

	component, err := c.resolveComponentLegacy(ctx, input, true)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentLegacyIdentity),
		)
		return err
	}
	observed.RecordStep(op+".current_resolved", "deleted component resolved for restore", nil,
		zap.String("component_id", component.ID.String()), zap.String("component_identity", component.Identity))
	if err = c.componentRepository.Restore(ctx, component.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_id", component.ID.String()),
			zap.String("component_identity", component.Identity),
		)
		return err
	}
	observed.RecordStep(op+".persisted", "component restored", nil,
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
func (c *ComponentLegacy) HardDelete(ctx context.Context, input ComponentLegacyIdentityInput) (err error) {
	const op = "component.hard_delete"
	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()
	ctx, observed := c.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)
	observed.RecordStep(op+".input_received", "component state change input received", nil,
		zap.String("project_identity", input.ProjectIdentity), zap.String("component_identity", input.ComponentLegacyIdentity))

	component, err := c.resolveComponentLegacy(ctx, input, true)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_identity", input.ComponentLegacyIdentity),
		)
		return err
	}
	observed.RecordStep(op+".current_resolved", "deleted component resolved for hard delete", nil,
		zap.String("component_id", component.ID.String()), zap.String("component_identity", component.Identity))
	if err = c.componentRepository.HardDelete(ctx, component.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("component_id", component.ID.String()),
			zap.String("component_identity", component.Identity),
		)
		return err
	}
	observed.RecordStep(op+".persisted", "component hard deleted", nil,
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
func (c *ComponentLegacy) Count(ctx context.Context, input ListComponentsLegacyInput) (count int64, err error) {
	const op = "component.count"
	ctx, cancel := context.WithTimeout(ctx, componentOperationTimeout)
	defer cancel()
	ctx, observed := c.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateListInput(&input.ProjectIdentity, input.FolderIdentity, input.ComponentType); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}
	observed.RecordStep(op+".input_validated", "component count input validated", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		zap.String("component_type", dereferenceComponentType(input.ComponentType)))
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}
	observed.RecordStep(op+".project_resolved", "project resolved for component count", nil,
		zap.String("project_id", project.ID.String()), zap.String("project_identity", project.Identity))
	filter := ports.ComponentsLegacyFilter{ProjectID: project.ID, ComponentType: input.ComponentType}
	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		)
		return 0, err
	}
	observed.RecordStep(op+".folder_filter_resolved", "component folder filter resolved", nil,
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)))
	count, err = c.componentRepository.Count(ctx, filter)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}
	observed.RecordStep(op+".result_loaded", "components counted", nil,
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		zap.Int64("count", count),
	)
	return count, nil
}
