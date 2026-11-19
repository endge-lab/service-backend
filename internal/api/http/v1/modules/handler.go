package modules

import (
	"net/url"
	"strings"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	usecase UseCase
}

func NewHandler(usecase UseCase) *Handler {
	return &Handler{usecase: usecase}
}

// @Summary Каталог модулей
// @Description Глобальный каталог активных модулей Backend. Администратор платформы может запросить только удалённые модули.
// @ID listModules
// @Tags Модули
// @Param deleted query boolean false "Показать только удалённые модули"
// @Success 200 {object} ListResponse "Глобальный каталог"
// @Failure 400 {object} shared.ErrorResponse "Некорректный фильтр"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Архив доступен администратору платформы"
// @Security BearerAuth
// @Router /api/v1/modules [get]
func (h *Handler) list(c *fiber.Ctx) error {
	deleted, err := shared.OptionalBoolQuery(c, "deleted", false)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	items, err := h.usecase.List(c.UserContext(), deleted)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	if items == nil {
		items = []entities.Module{}
	}
	return c.JSON(ListResponse{Items: items, Total: len(items)})
}

// @Summary Модуль и список версий
// @Description Метаданные модуля и все его опубликованные версии.
// @ID getModule
// @Tags Модули
// @Header 200 {string} ETag "Ревизия метаданных"
// @Param identity path string true "Идентификатор модуля"
// @Success 200 {object} entities.ModuleDetail "Карточка"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 404 {object} shared.ErrorResponse "Объект не найден"
// @Security BearerAuth
// @Router /api/v1/modules/{identity} [get]
func (h *Handler) detail(c *fiber.Ctx) error {
	result, err := h.usecase.Detail(c.UserContext(), c.Params("identity"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	if result.Versions == nil {
		result.Versions = []entities.ModuleVersion{}
	}
	setRevision(c, result.Revision)
	return c.JSON(result)
}

// @Summary Создать модуль и первую версию
// @ID createModule
// @Tags Модули
// @Description Операция администратора платформы. Каталог общий для всех рабочих пространств. Идентификатор и версия берутся из manifest.json.
// @Accept multipart/form-data
// @Header 201 {string} ETag "Ревизия метаданных"
// @Param identity formData string false "Ожидаемый идентификатор из manifest.json для совместимости со старыми клиентами"
// @Param name formData string true "Название"
// @Param description formData string false "Описание"
// @Param archive formData file true "ZIP сборки (не более 15 МиБ)"
// @Success 201 {object} entities.ModuleDetail "Модуль создан"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуется администратор платформы"
// @Failure 409 {object} shared.ErrorResponse "Конфликт версии, identity или ревизии"
// @Security BearerAuth
// @Router /api/v1/modules [post]
func (h *Handler) create(c *fiber.Ctx) error {
	data, err := h.readUpload(c)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	value := entities.Module{Identity: c.FormValue("identity"), Name: c.FormValue("name"), Description: c.FormValue("description")}
	result, err := h.usecase.Create(c.UserContext(), value, data)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	setRevision(c, result.Revision)
	return c.Status(fiber.StatusCreated).JSON(result)
}

// @Summary Изменить название или описание
// @ID patchModule
// @Tags Модули
// @Description Операция администратора платформы. Каталог общий для всех рабочих пространств.
// @Accept json
// @Header 200 {string} ETag "Новая ревизия метаданных"
// @Param identity path string true "Идентификатор модуля"
// @Param If-Match header string true "Ревизия метаданных"
// @Param request body PatchRequest true "Изменяемые поля"
// @Success 200 {object} entities.Module "Метаданные"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуется администратор платформы"
// @Failure 404 {object} shared.ErrorResponse "Объект не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт версии, identity или ревизии"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Security BearerAuth
// @Router /api/v1/modules/{identity} [patch]
func (h *Handler) patch(c *fiber.Ctx) error {
	version, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	var request PatchRequest
	if err := c.BodyParser(&request); err != nil || (request.Name == nil && request.Description == nil) {
		return respond.RespondDomainError(c, nil, domainerrors.InvalidInput("module_metadata_invalid", "Provide a name or description"))
	}
	result, err := h.usecase.Patch(c.UserContext(), c.Params("identity"), int64(version), request.Name, request.Description)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	setRevision(c, result.Revision)
	return c.JSON(result)
}

// @Summary Перенести модуль в архив
// @Description Убирает модуль из каталога. Ранее опубликованные версии остаются доступны по точным ссылкам; существующие привязки сохраняются.
// @ID deleteModule
// @Tags Модули
// @Header 200 {string} ETag "Новая ревизия метаданных"
// @Param identity path string true "Идентификатор модуля"
// @Param If-Match header string true "Ревизия метаданных"
// @Success 200 {object} entities.Module "Модуль в архиве"
// @Failure 400 {object} shared.ErrorResponse "Некорректная ревизия"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуется администратор платформы"
// @Failure 404 {object} shared.ErrorResponse "Модуль не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния или ревизии"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Security BearerAuth
// @Router /api/v1/modules/{identity} [delete]
func (h *Handler) softDelete(c *fiber.Ctx) error {
	return h.setDeleted(c, true)
}

// @Summary Восстановить модуль из архива
// @Description Возвращает модуль и его опубликованные версии в глобальный каталог.
// @ID restoreModule
// @Tags Модули
// @Header 200 {string} ETag "Новая ревизия метаданных"
// @Param identity path string true "Идентификатор модуля"
// @Param If-Match header string true "Ревизия метаданных"
// @Success 200 {object} entities.Module "Восстановленный модуль"
// @Failure 400 {object} shared.ErrorResponse "Некорректная ревизия"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуется администратор платформы"
// @Failure 404 {object} shared.ErrorResponse "Модуль не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния или ревизии"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Security BearerAuth
// @Router /api/v1/modules/{identity}/restore [post]
func (h *Handler) restore(c *fiber.Ctx) error {
	return h.setDeleted(c, false)
}

func (h *Handler) setDeleted(c *fiber.Ctx, deleted bool) error {
	revision, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	result, err := h.usecase.SetDeleted(c.UserContext(), c.Params("identity"), int64(revision), deleted)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	setRevision(c, result.Revision)
	return c.JSON(result)
}

// @Summary Опубликовать версию модуля
// @ID publishModuleVersion
// @Tags Модули
// @Description Операция администратора платформы. Каталог общий для всех рабочих пространств.
// @Accept multipart/form-data
// @Param identity path string true "Идентификатор модуля"
// @Param archive formData file true "ZIP сборки (не более 15 МиБ)"
// @Success 201 {object} entities.ModuleVersion "Опубликованная версия"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуется администратор платформы"
// @Failure 404 {object} shared.ErrorResponse "Объект не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт версии, identity или ревизии"
// @Security BearerAuth
// @Router /api/v1/modules/{identity}/versions [post]
func (h *Handler) publish(c *fiber.Ctx) error {
	data, err := h.readUpload(c)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	result, err := h.usecase.Publish(c.UserContext(), c.Params("identity"), data)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

// @Summary Метаданные версии модуля
// @Description Возвращает manifest, SHA-256 ZIP и адреса файлов для загрузки в браузере.
// @ID getModuleVersion
// @Tags Модули
// @Param identity path string true "Идентификатор модуля"
// @Param version path string true "Точная версия SemVer"
// @Success 200 {object} VersionResponse "Manifest, SHA-256 и адреса файлов"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 404 {object} shared.ErrorResponse "Объект не найден"
// @Security BearerAuth
// @Router /api/v1/modules/{identity}/versions/{version} [get]
func (h *Handler) version(c *fiber.Ctx) error {
	result, err := h.readVersion(c, false)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	base := c.Path()[:strings.LastIndexByte(c.Path(), '/')] + "/" + url.PathEscape(result.Version)
	assets := AssetURLs{JS: base + "/assets/module.js"}
	if result.HasCSS {
		assets.CSS = base + "/assets/module.css"
	}
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.JSON(VersionResponse{ModuleVersion: result, ModuleArchived: result.ModuleArchived, ArchiveURL: base + "/archive", Assets: assets})
}

// @Summary Скачать оригинальный ZIP
// @Description Возвращает неизменяемый оригинальный ZIP для точной версии.
// @ID downloadModuleArchive
// @Tags Модули
// @Produce application/zip
// @Param identity path string true "Идентификатор модуля"
// @Param version path string true "Точная версия SemVer"
// @Success 200 {file} file "ZIP"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 404 {object} shared.ErrorResponse "Объект не найден"
// @Security BearerAuth
// @Router /api/v1/modules/{identity}/versions/{version}/archive [get]
func (h *Handler) archive(c *fiber.Ctx) error {
	result, err := h.readVersion(c, true)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Attachment(result.Manifest.Identity + "-" + result.Version + ".zip")
	return sendFile(c, "application/zip", result.Archive)
}

// @Summary Файл сборки модуля
// @Description Загружает JavaScript или CSS точной опубликованной версии.
// @ID getModuleAsset
// @Tags Модули
// @Produce text/javascript,text/css
// @Param identity path string true "Идентификатор модуля"
// @Param version path string true "Точная версия SemVer"
// @Param file path string true "module.js или module.css" Enums(module.js,module.css)
// @Success 200 {file} file "JavaScript или CSS"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 404 {object} shared.ErrorResponse "Объект не найден"
// @Security BearerAuth
// @Router /api/v1/modules/{identity}/versions/{version}/assets/{file} [get]
func (h *Handler) asset(c *fiber.Ctx) error {
	result, err := h.readVersion(c, true)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	name := c.Params("file")
	data, err := h.usecase.ReadAsset(result.Archive, name)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	if name == "module.css" {
		return sendFile(c, "text/css; charset=utf-8", data)
	}
	return sendFile(c, "text/javascript; charset=utf-8", data)
}

func (h *Handler) readVersion(c *fiber.Ctx, archive bool) (entities.ModuleVersion, error) {
	version, err := shared.PathParam(c, "version")
	if err != nil {
		return entities.ModuleVersion{}, err
	}
	return h.usecase.Version(c.UserContext(), c.Params("identity"), version, archive)
}

func sendFile(c *fiber.Ctx, contentType string, data []byte) error {
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderContentType, contentType)
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(data)
}
