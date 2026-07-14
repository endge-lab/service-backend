package converters

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

const converterOperationTimeout = 15 * time.Second

type Converter struct {
	converterRepository ports.ConvertersRepository
	folderRepository    ports.FoldersRepository
	projectRepository   ports.ProjectsRepository
	observed            shared.ObservedUseCase
}

type ConverterParams struct {
	ConverterRepository ports.ConvertersRepository
	FolderRepository    ports.FoldersRepository
	ProjectRepository   ports.ProjectsRepository
	Tracer              trace.Tracer
	Logger              *zap.Logger
	Metrics             *shared.UseCaseMetrics
}

func NewConverterService(params ConverterParams) *Converter {
	return &Converter{
		converterRepository: params.ConverterRepository,
		folderRepository:    params.FolderRepository,
		projectRepository:   params.ProjectRepository,
		observed:            shared.NewObservedUseCase(params.Tracer, params.Logger, params.Metrics),
	}
}

// Create создает конвертер в указанной папке проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует input, разрешает project и folder converters, проверяет identity и создает конвертер.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Converter) Create(ctx context.Context, input adapters.CreateConverterInput) (result *entities.Converter, err error) {
	const op = "converter.create"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeAndValidateCreateInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}
	folder, err := c.folderRepository.GetByIdentity(ctx, &project.ID, entities.FolderEntityTypeConverters, input.FolderIdentity)
	if err != nil {
		return nil, apperrors.InvalidInput("folder_entity_type_mismatch", "folder must belong to the project and have converters entity type")
	}
	exists, err := c.converterRepository.ExistsByIdentity(ctx, project.ID, input.Identity)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperrors.Conflict("identity_conflict", "converter identity already exists")
	}
	result, err = c.converterRepository.Create(ctx, &entities.Converter{ProjectID: project.ID, FolderID: folder.ID, Identity: input.Identity, DisplayName: input.DisplayName, Description: input.Description, ConverterType: input.ConverterType, Source: input.Source, IsSystem: input.IsSystem, Meta: input.Meta, Active: input.Active})
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}
	return result, nil
}

// Update обновляет конвертер в указанной папке проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует input, получает project, существующий конвертер и folder converters, затем обновляет конвертер.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Converter) Update(ctx context.Context, input adapters.UpdateConverterInput) (result *entities.Converter, err error) {
	const op = "converter.update"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeAndValidateUpdateInput(&input); err != nil {
		observed.Logger().Warn(op, zap.Error(err))
		return nil, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}
	current, err := c.converterRepository.GetByIdentity(ctx, project.ID, input.ConverterIdentity)
	if err != nil {
		return nil, err
	}
	folder, err := c.folderRepository.GetByIdentity(ctx, &project.ID, entities.FolderEntityTypeConverters, input.FolderIdentity)
	if err != nil {
		return nil, apperrors.InvalidInput("folder_entity_type_mismatch", "folder must belong to the project and have converters entity type")
	}
	result, err = c.converterRepository.Update(ctx, &entities.Converter{ID: current.ID, ProjectID: current.ProjectID, FolderID: folder.ID, Identity: current.Identity, DisplayName: input.DisplayName, Description: input.Description, ConverterType: input.ConverterType, Source: input.Source, IsSystem: input.IsSystem, Meta: input.Meta, Active: input.Active, DeletedAt: current.DeletedAt, CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt})
	if err != nil {
		observed.Logger().Error(op, zap.Error(err))
		return nil, err
	}
	return result, nil
}

// GetByIdentity возвращает активный конвертер по identity в пределах проекта.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует identities, получает project и запрашивает активный конвертер.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Converter) GetByIdentity(ctx context.Context, input adapters.GetConverterInput) (result *entities.Converter, err error) {
	const op = "converter.get_by_identity"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeAndValidateIdentityInput(&input.ProjectIdentity, &input.ConverterIdentity); err != nil {
		return nil, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}
	result, err = c.converterRepository.GetByIdentity(ctx, project.ID, input.ConverterIdentity)
	return result, err
}

// List возвращает активные конвертеры проекта с учетом фильтра.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует фильтр, получает project и optional folder converters, затем запрашивает список.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Converter) List(ctx context.Context, input adapters.ListConvertersInput) (result []*entities.Converter, err error) {
	const op = "converter.list"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeAndValidateListInput(&input.ProjectIdentity, input.FolderIdentity); err != nil {
		return nil, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return nil, err
	}
	filter := ports.ConvertersFilter{ProjectID: project.ID}
	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		return nil, err
	}
	result, err = c.converterRepository.List(ctx, filter)
	return result, err
}

// SoftDelete выполняет мягкое удаление конвертера.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Разрешает converter identity в пределах проекта и выполняет мягкое удаление.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Converter) SoftDelete(ctx context.Context, input adapters.ConverterIdentityInput) (err error) {
	const op = "converter.soft_delete"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	converter, err := c.resolveConverter(ctx, input, false)
	if err != nil {
		return err
	}
	return c.converterRepository.SoftDelete(ctx, converter.ID)
}

// Restore восстанавливает мягко удаленный конвертер.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Разрешает converter identity с учетом soft-delete и восстанавливает конвертер.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Converter) Restore(ctx context.Context, input adapters.ConverterIdentityInput) (err error) {
	const op = "converter.restore"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	converter, err := c.resolveConverter(ctx, input, true)
	if err != nil {
		return err
	}
	return c.converterRepository.Restore(ctx, converter.ID)
}

// HardDelete физически удаляет конвертер, если он не системный.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Разрешает converter identity, запрещает удаление системного конвертера и физически удаляет запись.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Converter) HardDelete(ctx context.Context, input adapters.ConverterIdentityInput) (err error) {
	const op = "converter.hard_delete"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	converter, err := c.resolveConverter(ctx, input, true)
	if err != nil {
		return err
	}
	if converter.IsSystem {
		return apperrors.Conflict("system_converter_delete_forbidden", "system converter cannot be hard deleted")
	}
	return c.converterRepository.HardDelete(ctx, converter.ID)
}

// Count возвращает количество активных конвертеров с учетом фильтра.
//
// Параметры:
//
//	ctx - контекст выполнения
//
// Что делает функция:
//
//	Валидирует фильтр, разрешает project и optional folder converters, затем подсчитывает конвертеры.
//
// Возвращаемые значения:
//
//	Результат операции или ошибка, возникшая при ее выполнении.
func (c *Converter) Count(ctx context.Context, input adapters.ListConvertersInput) (count int64, err error) {
	const op = "converter.count"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeAndValidateListInput(&input.ProjectIdentity, input.FolderIdentity); err != nil {
		return 0, err
	}
	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		return 0, err
	}
	filter := ports.ConvertersFilter{ProjectID: project.ID}
	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		return 0, err
	}
	return c.converterRepository.Count(ctx, filter)
}
