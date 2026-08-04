package backup

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает HTTP-операции страховочных snapshot backups.
type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт backup HTTP-обработчик.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// Create сохраняет manual backup текущего workspace.
// @Summary Создать backup workspace
// @Description Сохраняет бессрочный переносимый snapshot текущего workspace с опциональным описанием.
// @ID createDomainBackup
// @Tags Backups
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param request body CreateRequest false "Опциональное описание backup"
// @Success 201 {object} Response "Backup создан"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуются права workspace admin"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain/backups [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	request := CreateRequest{}
	var err error
	if len(strings.TrimSpace(string(c.Body()))) > 0 {
		request, err = shared.DecodeAndValidate[CreateRequest](c, h.validator)
		if err != nil {
			return respond.WriteErrorResponse(c, err)
		}
	}
	value, err := h.usecase.Create(c.UserContext(), request.Description)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Location("/api/v1/domain/backups/" + value.ID)
	return c.Status(fiber.StatusCreated).JSON(newResponse(*value))
}

// List возвращает metadata доступных backups без JSON-содержимого.
// @Summary Получить backups workspace
// @Description Возвращает manual и pre-import backups без тяжёлого snapshot data.
// @ID listDomainBackups
// @Tags Backups
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param kind query string false "Тип backup" Enums(manual,pre_import)
// @Param limit query int false "Размер страницы" default(100) minimum(1) maximum(500)
// @Param offset query int false "Смещение" default(0) minimum(0)
// @Success 200 {object} ListResponse "Список backups"
// @Failure 400 {object} shared.ErrorResponse "Некорректный фильтр"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуются права workspace admin"
// @Security BearerAuth
// @Router /api/v1/domain/backups [get]
func (h *Handler) List(c *fiber.Ctx) error {
	limit, offset, err := shared.LimitOffset(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	values, err := h.usecase.List(c.UserContext(), strings.TrimSpace(c.Query("kind")), limit, offset)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items := make([]Response, 0, len(values))
	for _, value := range values {
		items = append(items, newResponse(value))
	}
	return c.JSON(ListResponse{Items: items, Total: len(items)})
}

// Get возвращает metadata backup по UUID или alias last.
// @Summary Получить backup
// @Description Возвращает metadata backup; id=last выбирает последний успешно сохранённый backup.
// @ID getDomainBackup
// @Tags Backups
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param id path string true "UUID backup или last"
// @Success 200 {object} Response "Backup"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуются права workspace admin"
// @Failure 404 {object} shared.ErrorResponse "Backup не найден"
// @Security BearerAuth
// @Router /api/v1/domain/backups/{id} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	value, err := h.usecase.Get(c.UserContext(), c.Params("id"), false)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(newResponse(*value))
}

// Export возвращает portable JSON backup inline либо как attachment.
// @Summary Экспортировать backup
// @Description Возвращает backup по UUID или alias last в каноническом workspace snapshot формате.
// @ID exportDomainBackup
// @Tags Backups
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param id path string true "UUID backup или last"
// @Param download query bool false "Скачать JSON как файл" default(false)
// @Success 200 {object} ExportResponse "Workspace snapshot"
// @Header 200 {string} ETag "Checksum backup"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуются права workspace admin"
// @Failure 404 {object} shared.ErrorResponse "Backup не найден"
// @Security BearerAuth
// @Router /api/v1/domain/backups/{id}/export [get]
func (h *Handler) Export(c *fiber.Ctx) error {
	value, err := h.usecase.Get(c.UserContext(), c.Params("id"), true)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Type("json")
	c.Set(fiber.HeaderETag, `"`+value.Checksum+`"`)
	download, err := shared.OptionalBoolQuery(c, "download", false)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	if download {
		c.Attachment("backup-" + value.ID + ".json")
	}
	return c.Send(value.Data)
}

// Archive потоково формирует ZIP со всеми доступными backups workspace.
// @Summary Скачать backups в ZIP
// @Description Формирует ZIP с manifest.json и snapshots; опционально фильтрует по kind.
// @ID archiveDomainBackups
// @Tags Backups
// @Produce application/zip
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param kind query string false "Тип backup" Enums(manual,pre_import)
// @Success 200 {file} binary "ZIP архив"
// @Failure 400 {object} shared.ErrorResponse "Некорректный фильтр"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуются права workspace admin"
// @Security BearerAuth
// @Router /api/v1/domain/backups/archive [get]
func (h *Handler) Archive(c *fiber.Ctx) error {
	values, err := h.usecase.Archive(c.UserContext(), strings.TrimSpace(c.Query("kind")))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	manifest := map[string]any{"schemaVersion": 1, "createdAt": time.Now().UTC(), "backups": archiveManifest(values)}
	c.Type("zip")
	c.Attachment("workspace-backups-" + time.Now().UTC().Format("20060102-150405") + ".zip")
	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		archive := zip.NewWriter(writer)
		if entry, createErr := archive.Create("manifest.json"); createErr == nil {
			_ = json.NewEncoder(entry).Encode(manifest)
		}
		for _, value := range values {
			name := filepath.Join("backups", fmt.Sprintf("%s_%s.json", value.CreatedAt.UTC().Format("20060102T150405Z"), value.ID))
			if entry, createErr := archive.Create(name); createErr == nil {
				_, _ = entry.Write(value.Data)
			}
		}
		_ = archive.Close()
	})
	return nil
}

func archiveManifest(values []entities.SnapshotBackup) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]any{"id": value.ID, "kind": value.Kind, "description": value.Description, "checksum": value.Checksum, "createdAt": value.CreatedAt})
	}
	return result
}
