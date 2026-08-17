package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// PathParam декодирует отдельный path-параметр после сопоставления маршрута.
// Это сохраняет закодированные разделители внутри одного сегмента маршрута.
func PathParam(c *fiber.Ctx, name string) (string, error) {
	value, err := url.PathUnescape(c.Params(name))
	if err != nil {
		return "", domainerrors.WithDetails(
			domainerrors.InvalidInput("path_parameter_invalid", name+" must be URL-encoded"),
			map[string]any{"field": name},
		)
	}
	return value, nil
}

// OptionalBoolQuery строго разбирает необязательный boolean query-параметр.
func OptionalBoolQuery(c *fiber.Ctx, name string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, domainerrors.WithDetails(domainerrors.InvalidInput("query_boolean_invalid", name+" must be boolean"), map[string]any{"field": name})
	}
	return value, nil
}

// LimitOffset строго разбирает стандартную пагинацию list endpoint.
func LimitOffset(c *fiber.Ctx) (int, int, error) {
	limit, offset := c.QueryInt("limit", 100), c.QueryInt("offset", 0)
	if limit < 1 || limit > 500 {
		return 0, 0, domainerrors.InvalidInput("limit_invalid", "limit must be between 1 and 500")
	}
	if offset < 0 {
		return 0, 0, domainerrors.InvalidInput("offset_invalid", "offset must be non-negative")
	}
	return limit, offset, nil
}

// SafeAttachmentName удаляет управляющие и разделительные символы из имени HTTP-вложения.
func SafeAttachmentName(value, fallback string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, strings.TrimSpace(value))
	value = strings.Trim(value, ".-")
	if value == "" {
		return fallback
	}
	return value
}

type CreateDocumentRequest struct {
	Identity       string         `json:"identity" validate:"required,max=160" example:"main"`
	DisplayName    string         `json:"displayName" validate:"required,max=255" example:"Основной объект"`
	Description    *string        `json:"description,omitempty" example:"Описание объекта"`
	FolderIdentity *string        `json:"folderIdentity,omitempty" example:"root-projects"`
	ManagedBy      string         `json:"managedBy,omitempty" validate:"omitempty,oneof=user system integration" example:"user" enums:"user,system,integration"`
	ManagedByID    *string        `json:"managedById,omitempty" example:"endge-core"`
	Meta           map[string]any `json:"meta,omitempty"`
	Active         *bool          `json:"active,omitempty" example:"true"`
}

type PatchDocumentRequest struct {
	Identity       *string        `json:"identity,omitempty" validate:"omitempty,max=160" example:"main"`
	DisplayName    *string        `json:"displayName,omitempty" validate:"omitempty,max=255" example:"Основной объект"`
	Description    *string        `json:"description,omitempty" example:"Описание объекта"`
	FolderIdentity *string        `json:"folderIdentity,omitempty" example:"root-projects"`
	ManagedBy      *string        `json:"managedBy,omitempty" validate:"omitempty,oneof=user system integration" example:"user" enums:"user,system,integration"`
	ManagedByID    *string        `json:"managedById,omitempty" example:"endge-core"`
	Meta           map[string]any `json:"meta,omitempty"`
	Active         *bool          `json:"active,omitempty" example:"true"`
}

type DocumentMetadata struct {
	ID             string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000" format:"uuid"`
	Identity       string          `json:"identity" example:"main"`
	DisplayName    string          `json:"displayName" example:"Основной объект"`
	Description    *string         `json:"description,omitempty" example:"Описание объекта"`
	FolderIdentity *string         `json:"folderIdentity,omitempty" example:"root-projects"`
	ManagedBy      string          `json:"managedBy" example:"user" enums:"user,system,integration"`
	ManagedByID    *string         `json:"managedById,omitempty" example:"endge-core"`
	Meta           json.RawMessage `json:"meta" swaggertype:"object"`
	Active         bool            `json:"active" example:"true"`
	DeletedAt      *time.Time      `json:"deletedAt,omitempty" example:"2026-08-04T11:00:00Z" format:"date-time"`
	Revision       int             `json:"revision" example:"3"`
	CreatedBy      entities.Actor  `json:"createdBy"`
	UpdatedBy      entities.Actor  `json:"updatedBy"`
	CreatedAt      time.Time       `json:"createdAt" example:"2026-08-04T10:00:00Z" format:"date-time"`
	UpdatedAt      time.Time       `json:"updatedAt" example:"2026-08-04T10:05:00Z" format:"date-time"`
}

// ErrorResponse переиспользует единый внешний envelope ошибок в Swagger-схемах.
type ErrorResponse = respond.ErrorResponse

// DecodeAndValidate декодирует JSON в транспортную модель, запрещает неизвестные поля
// и применяет декларативные validate-теги.
func DecodeAndValidate[T any](c *fiber.Ctx, validator appvalidator.Validator) (T, error) {
	var contract T
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return contract, domainerrors.WithDetails(respond.ErrInvalidBody, map[string]any{"reason": err.Error()})
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return contract, domainerrors.WithDetails(respond.ErrInvalidBody, map[string]any{"reason": err.Error()})
	}
	if validator != nil {
		if err := validator.Validate(contract); err != nil {
			if validationErr, ok := appvalidator.AsValidationErr(err); ok {
				return contract, domainerrors.WithDetails(respond.ErrValidationError, map[string]any{"fields": validationErr.Fields})
			}
			return contract, err
		}
	}
	return contract, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("тело запроса должно содержать одно JSON-значение")
		}
		return err
	}
	return nil
}

// IfMatch извлекает обязательную положительную revision из HTTP-заголовка.
func IfMatch(c *fiber.Ctx) (int, error) {
	raw := strings.TrimSpace(c.Get("If-Match"))
	if raw == "" {
		return 0, domainerrors.New("precondition_required", "If-Match header is required", 428)
	}
	raw = strings.TrimPrefix(raw, "W/")
	raw = strings.Trim(raw, "\"")
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, domainerrors.InvalidInput("if_match_invalid", "If-Match must contain a positive revision")
	}
	return value, nil
}

// ETag формирует strong ETag из revision документа.
func ETag(revision int) string { return `"` + strconv.Itoa(revision) + `"` }

// IfNoneMatch сообщает, совпадает ли хотя бы один клиентский ETag с текущим.
// Для GET используется weak comparison, поэтому W/"tag" и "tag" эквивалентны.
func IfNoneMatch(c *fiber.Ctx, currentETag string) bool {
	currentETag = normalizeEntityTag(currentETag)
	if currentETag == "" {
		return false
	}
	for _, value := range strings.Split(c.Get(fiber.HeaderIfNoneMatch), ",") {
		value = strings.TrimSpace(value)
		if value == "*" || normalizeEntityTag(value) == currentETag {
			return true
		}
	}
	return false
}

func normalizeEntityTag(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "W/")
	return strings.Trim(value, `"`)
}

// DocumentMap объединяет resource payload и серверные метаданные документа.
func DocumentMap(doc entities.Document) (map[string]any, error) {
	result := map[string]any{}
	if err := json.Unmarshal(doc.Data, &result); err != nil {
		return nil, domainerrors.Internal("transport_mapping_failed", "Не удалось сформировать ответ")
	}
	result["id"] = doc.ID
	result["identity"] = doc.Identity
	result["displayName"] = doc.DisplayName
	result["description"] = doc.Description
	result["folderIdentity"] = doc.FolderIdentity
	result["managedBy"] = doc.ManagedBy
	result["managedById"] = doc.ManagedByID
	result["meta"] = json.RawMessage(doc.Meta)
	result["active"] = doc.Active
	result["deletedAt"] = doc.DeletedAt
	result["revision"] = doc.Revision
	result["createdBy"] = doc.CreatedBy
	result["updatedBy"] = doc.UpdatedBy
	result["createdAt"] = doc.CreatedAt
	result["updatedAt"] = doc.UpdatedAt
	return result, nil
}

// DecodeDocument безопасно преобразует доменный документ в ресурсный ответ.
func DecodeDocument[T any](doc entities.Document) (T, error) {
	value, err := DocumentMap(doc)
	if err != nil {
		var zero T
		return zero, err
	}
	return DecodeValue[T](value)
}

// DecodeValue преобразует значение между типизированными транспортными моделями без panic.
func DecodeValue[T any](value any) (T, error) {
	var result T
	raw, err := json.Marshal(value)
	if err != nil {
		return result, domainerrors.Internal("transport_mapping_failed", "Не удалось сформировать ответ")
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return result, domainerrors.Internal("transport_mapping_failed", "Не удалось сформировать ответ")
	}
	return result, nil
}

// MapValues безопасно преобразует список application-значений в HTTP-модели.
func MapValues[Source, Target any](values []Source, mapper func(Source) (Target, error)) ([]Target, error) {
	items := make([]Target, 0, len(values))
	for _, value := range values {
		item, err := mapper(value)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// ParseDocumentFilter читает и проверяет общие list query-параметры.
func ParseDocumentFilter(c *fiber.Ctx) (ports.DocumentFilter, error) {
	filter := ports.DocumentFilter{
		IncludeDeleted: c.QueryBool("includeDeleted", false),
		Limit:          c.QueryInt("limit", 100),
		Offset:         c.QueryInt("offset", 0),
	}
	if filter.Limit < 1 || filter.Limit > 500 {
		return filter, domainerrors.InvalidInput("limit_invalid", "limit must be between 1 and 500")
	}
	if filter.Offset < 0 {
		return filter, domainerrors.InvalidInput("offset_invalid", "offset must be non-negative")
	}
	if value := strings.TrimSpace(c.Query("folderIdentity")); value != "" {
		filter.FolderIdentity = &value
	}
	if raw := strings.TrimSpace(c.Query("active")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return filter, domainerrors.InvalidInput("active_invalid", "active must be boolean")
		}
		filter.Active = &value
	}
	return filter, nil
}

// DefaultString возвращает fallback для пустой строки.
func DefaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
