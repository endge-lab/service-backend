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
func (c *Converter) Create(ctx context.Context, input adapters.CreateConverterInput) (result *adapters.ConverterWithFolder, err error) {
	const op = "converter.create"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
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
	exists, err := c.converterRepository.ExistsByIdentity(ctx, project.ID, input.Identity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", input.Identity),
		)
		return nil, err
	}
	if exists {
		err = apperrors.Conflict("identity_conflict", "converter identity already exists")
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", input.Identity),
		)
		return nil, err
	}
	converterResult, err := c.converterRepository.Create(ctx, &entities.Converter{ProjectID: project.ID, FolderID: folder.ID, Identity: input.Identity, DisplayName: input.DisplayName, Description: input.Description, ConverterType: input.ConverterType, Source: input.Source, IsSystem: input.IsSystem, Meta: input.Meta, Active: input.Active})
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", input.Identity),
		)
		return nil, err
	}
	observed.Logger().Debug("converter created",
		zap.String("converter_id", converterResult.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("converter_identity", converterResult.Identity),
		zap.String("folder_id", folder.ID.String()),
		zap.String("folder_identity", folder.Identity),
		zap.String("converter_type", converterResult.ConverterType),
	)
	return converterWithFolder(converterResult, folder.Identity), nil
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
func (c *Converter) Update(ctx context.Context, input adapters.UpdateConverterInput) (result *adapters.ConverterWithFolder, err error) {
	const op = "converter.update"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
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
	current, err := c.converterRepository.GetByIdentity(ctx, project.ID, input.ConverterIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", input.ConverterIdentity),
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
	converterResult, err := c.converterRepository.Update(ctx, &entities.Converter{ID: current.ID, ProjectID: current.ProjectID, FolderID: folder.ID, Identity: current.Identity, DisplayName: input.DisplayName, Description: input.Description, ConverterType: input.ConverterType, Source: input.Source, IsSystem: input.IsSystem, Meta: input.Meta, Active: input.Active, DeletedAt: current.DeletedAt, CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt})
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", current.Identity),
		)
		return nil, err
	}
	observed.Logger().Debug("converter updated",
		zap.String("converter_id", converterResult.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("converter_identity", converterResult.Identity),
		zap.String("folder_id", folder.ID.String()),
		zap.String("folder_identity", folder.Identity),
		zap.String("converter_type", converterResult.ConverterType),
	)
	return converterWithFolder(converterResult, folder.Identity), nil
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
func (c *Converter) GetByIdentity(ctx context.Context, input adapters.GetConverterInput) (result *adapters.ConverterWithFolder, err error) {
	const op = "converter.get_by_identity"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeAndValidateIdentityInput(&input.ProjectIdentity, &input.ConverterIdentity); err != nil {
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
	converter, err := c.converterRepository.GetByIdentity(ctx, project.ID, input.ConverterIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", input.ConverterIdentity),
		)
		return nil, err
	}
	folder, err := c.folderRepository.GetByID(ctx, converter.FolderID)
	if err != nil {
		relationErr := apperrors.Internal("converter_folder_not_found", "converter references an unavailable folder")
		logOperationError(observed.Logger(), op, relationErr,
			zap.NamedError("repository_error", err),
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_id", converter.ID.String()),
			zap.String("converter_identity", converter.Identity),
			zap.String("folder_id", converter.FolderID.String()),
		)
		return nil, relationErr
	}
	observed.Logger().Debug("converter retrieved",
		zap.String("converter_id", converter.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("converter_identity", converter.Identity),
		zap.String("folder_id", converter.FolderID.String()),
		zap.String("folder_identity", folder.Identity),
	)
	return converterWithFolder(converter, folder.Identity), nil
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
func (c *Converter) List(ctx context.Context, input adapters.ListConvertersInput) (result []*adapters.ConverterWithFolder, err error) {
	const op = "converter.list"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()
	ctx, observed := c.observed.StartObservedOperation(ctx, op, nil, nil)
	defer observed.End(&err)
	if err = normalizeAndValidateListInput(&input.ProjectIdentity, input.FolderIdentity); err != nil {
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
	filter := ports.ConvertersFilter{ProjectID: project.ID}
	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		)
		return nil, err
	}
	converters, err := c.converterRepository.List(ctx, filter)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	if len(converters) == 0 {
		observed.Logger().Debug("converters listed",
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
			zap.Int("count", 0),
		)
		return make([]*adapters.ConverterWithFolder, 0), nil
	}
	folders, err := c.folderRepository.List(ctx, &project.ID, entities.FolderEntityTypeConverters)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return nil, err
	}
	result, err = converterWithFolders(converters, folders)
	if err != nil {
		fields := []zap.Field{
			zap.String("project_identity", input.ProjectIdentity),
			zap.Int("converter_count", len(converters)),
		}
		if converter := firstConverterWithUnavailableFolder(converters, folders); converter != nil {
			fields = append(fields,
				zap.String("converter_id", converter.ID.String()),
				zap.String("converter_identity", converter.Identity),
				zap.String("folder_id", converter.FolderID.String()),
			)
		}
		logOperationError(observed.Logger(), op, err, fields...)
		return nil, err
	}
	observed.Logger().Debug("converters listed",
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		zap.Int("count", len(result)),
	)
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
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", input.ConverterIdentity),
		)
		return err
	}
	if err = c.converterRepository.SoftDelete(ctx, converter.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_id", converter.ID.String()),
			zap.String("converter_identity", converter.Identity),
		)
		return err
	}
	observed.Logger().Debug("converter soft deleted",
		zap.String("converter_id", converter.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("converter_identity", converter.Identity),
	)
	return nil
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
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", input.ConverterIdentity),
		)
		return err
	}
	if err = c.converterRepository.Restore(ctx, converter.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_id", converter.ID.String()),
			zap.String("converter_identity", converter.Identity),
		)
		return err
	}
	observed.Logger().Debug("converter restored",
		zap.String("converter_id", converter.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("converter_identity", converter.Identity),
	)
	return nil
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
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_identity", input.ConverterIdentity),
		)
		return err
	}
	if converter.IsSystem {
		err = apperrors.Conflict("system_converter_delete_forbidden", "system converter cannot be hard deleted")
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_id", converter.ID.String()),
			zap.String("converter_identity", converter.Identity),
		)
		return err
	}
	if err = c.converterRepository.HardDelete(ctx, converter.ID); err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("converter_id", converter.ID.String()),
			zap.String("converter_identity", converter.Identity),
		)
		return err
	}
	observed.Logger().Debug("converter hard deleted",
		zap.String("converter_id", converter.ID.String()),
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("converter_identity", converter.Identity),
	)
	return nil
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
	filter := ports.ConvertersFilter{ProjectID: project.ID}
	filter.FolderID, err = c.resolveFolderID(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
			zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		)
		return 0, err
	}
	count, err = c.converterRepository.Count(ctx, filter)
	if err != nil {
		logOperationError(observed.Logger(), op, err,
			zap.String("project_identity", input.ProjectIdentity),
		)
		return 0, err
	}
	observed.Logger().Debug("converters counted",
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("folder_identity", dereferenceString(input.FolderIdentity)),
		zap.Int64("count", count),
	)
	return count, nil
}
