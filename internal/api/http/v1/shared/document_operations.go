package shared

import (
	"context"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// ListDocuments выполняет общий HTTP-сценарий получения списка документов.
func ListDocuments[T any](c *fiber.Ctx, operation func(context.Context, ports.DocumentFilter) ([]entities.Document, error), mapper func(entities.Document) (T, error)) error {
	filter, err := ParseDocumentFilter(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	values, err := operation(c.UserContext(), filter)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items := make([]T, 0, len(values))
	for _, value := range values {
		item, mapErr := mapper(value)
		if mapErr != nil {
			return respond.RespondDomainError(c, nil, mapErr)
		}
		items = append(items, item)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items), "limit": filter.Limit, "offset": filter.Offset})
}

// CreateDocument выполняет общий HTTP-сценарий создания документа.
func CreateDocument[Request, Response any](c *fiber.Ctx, validator appvalidator.Validator, operation func(context.Context, documents.CreateInput) (*entities.Document, error), mapper func(entities.Document) (Response, error)) error {
	_, err := DecodeAndValidate[Request](c, validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	input, err := documents.NewCreateInputJSON(c.Body())
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := operation(c.UserContext(), input)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return writeDocument(c, fiber.StatusCreated, *value, mapper)
}

// GetDocument выполняет общий HTTP-сценарий получения документа.
func GetDocument[Response any](c *fiber.Ctx, operation func(context.Context, string, bool) (*entities.Document, error), mapper func(entities.Document) (Response, error)) error {
	value, err := operation(c.UserContext(), c.Params("identity"), c.QueryBool("includeDeleted", false))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return writeDocument(c, fiber.StatusOK, *value, mapper)
}

// PatchDocument выполняет общий HTTP-сценарий конкурентного изменения документа.
func PatchDocument[Request, Response any](c *fiber.Ctx, validator appvalidator.Validator, operation func(context.Context, string, documents.PatchInput, int) (*entities.Document, error), mapper func(entities.Document) (Response, error)) error {
	expected, err := IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	_, err = DecodeAndValidate[Request](c, validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	input, err := documents.NewPatchInputJSON(c.Body())
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := operation(c.UserContext(), c.Params("identity"), input, expected)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return writeDocument(c, fiber.StatusOK, *value, mapper)
}

// MutateDocument выполняет DELETE или restore с обязательным If-Match.
func MutateDocument[Response any](c *fiber.Ctx, operation func(context.Context, string, int) (*entities.Document, error), mapper func(entities.Document) (Response, error)) error {
	expected, err := IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := operation(c.UserContext(), c.Params("identity"), expected)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return writeDocument(c, fiber.StatusOK, *value, mapper)
}

func writeDocument[Response any](c *fiber.Ctx, status int, document entities.Document, mapper func(entities.Document) (Response, error)) error {
	response, err := mapper(document)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, ETag(document.Revision))
	return c.Status(status).JSON(response)
}
