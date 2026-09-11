package facet

import (
	"context"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	revisionapi "github.com/endge-lab/service-backend/internal/api/http/v1/revision"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// List godoc
// @Summary Получить фасеты
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param includeDeleted query bool false "Включить удалённые фасеты"
// @Success 200 {object} FacetListResponse
// @Security BearerAuth
// @Router /api/v1/facets [get]
func (h *Handler) List(c *fiber.Ctx) error {
	includeDeleted, err := shared.OptionalBoolQuery(c, "includeDeleted", false)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	values, err := h.usecase.List(c.UserContext(), includeDeleted)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := shared.MapValues(values, NewFacetResponse)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// Create godoc
// @Summary Создать фасет
// @Tags Фасеты
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param request body CreateRequest true "Фасет"
// @Success 201 {object} FacetResponse
// @Security BearerAuth
// @Router /api/v1/facets [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Create(c.UserContext(), request.Input())
	return writeFacet(c, fiber.StatusCreated, value, err)
}

// Reorder godoc
// @Summary Изменить порядок фасетов
// @Tags Фасеты
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param request body ReorderRequest true "Полный порядок активных фасетов"
// @Success 200 {object} FacetListResponse
// @Security BearerAuth
// @Router /api/v1/facets/reorder [post]
func (h *Handler) Reorder(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[ReorderRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	values, err := h.usecase.Reorder(c.UserContext(), request.Input())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := shared.MapValues(values, NewFacetResponse)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// Get godoc
// @Summary Получить фасет
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param includeDeleted query bool false "Разрешить удалённый фасет"
// @Success 200 {object} FacetResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	identity, err := path(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Get(c.UserContext(), identity, c.QueryBool("includeDeleted", false))
	return writeFacet(c, fiber.StatusOK, value, err)
}

// Patch godoc
// @Summary Изменить фасет
// @Tags Фасеты
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param If-Match header string true "Текущая revision фасета"
// @Param request body PatchRequest true "Изменения фасета"
// @Success 200 {object} FacetResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity} [patch]
func (h *Handler) Patch(c *fiber.Ctx) error {
	identity, expected, err := identityAndRevision(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	request, err := shared.DecodeAndValidate[PatchRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Patch(c.UserContext(), identity, request.Input(), expected)
	return writeFacet(c, fiber.StatusOK, value, err)
}

// Delete godoc
// @Summary Мягко удалить фасет
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param If-Match header string true "Текущая revision фасета"
// @Success 200 {object} FacetResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	identity, expected, err := identityAndRevision(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Delete(c.UserContext(), identity, expected)
	return writeFacet(c, fiber.StatusOK, value, err)
}

// Restore godoc
// @Summary Восстановить фасет
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param If-Match header string true "Текущая revision фасета"
// @Success 200 {object} FacetResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error {
	identity, expected, err := identityAndRevision(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Restore(c.UserContext(), identity, expected)
	return writeFacet(c, fiber.StatusOK, value, err)
}

// ListDocuments godoc
// @Summary Получить документы фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param includeDeleted query bool false "Включить удалённые документы"
// @Param active query bool false "Фильтр активности"
// @Param limit query int false "Размер страницы"
// @Param offset query int false "Смещение"
// @Success 200 {object} FacetDocumentListResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents [get]
func (h *Handler) ListDocuments(c *fiber.Ctx) error {
	facetIdentity, err := path(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	filter, err := shared.ParseDocumentFilter(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	values, err := h.usecase.ListDocuments(c.UserContext(), facetIdentity, filter)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := shared.MapValues(values, NewFacetDocumentResponse)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items), "limit": filter.Limit, "offset": filter.Offset})
}

// CreateDocument godoc
// @Summary Создать документ фасета
// @Tags Фасеты
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param request body DocumentCreateRequest true "Документ фасета"
// @Success 201 {object} FacetDocumentResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents [post]
func (h *Handler) CreateDocument(c *fiber.Ctx) error {
	facetIdentity, err := path(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	request, err := shared.DecodeAndValidate[DocumentCreateRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.CreateDocument(c.UserContext(), facetIdentity, request.Input())
	return writeFacetDocument(c, fiber.StatusCreated, value, err)
}

// GetDocument godoc
// @Summary Получить документ фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param documentIdentity path string true "Identity документа внутри фасета"
// @Param includeDeleted query bool false "Разрешить удалённый документ"
// @Success 200 {object} FacetDocumentResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents/{documentIdentity} [get]
func (h *Handler) GetDocument(c *fiber.Ctx) error {
	facetIdentity, documentIdentity, err := documentPath(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.GetDocument(c.UserContext(), facetIdentity, documentIdentity, c.QueryBool("includeDeleted", false))
	return writeFacetDocument(c, fiber.StatusOK, value, err)
}

// PatchDocument godoc
// @Summary Изменить документ фасета
// @Tags Фасеты
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param documentIdentity path string true "Identity документа внутри фасета"
// @Param If-Match header string true "Текущая revision документа"
// @Param request body DocumentPatchRequest true "Изменения документа"
// @Success 200 {object} FacetDocumentResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents/{documentIdentity} [patch]
func (h *Handler) PatchDocument(c *fiber.Ctx) error {
	facetIdentity, documentIdentity, err := documentPath(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	expected, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	request, err := shared.DecodeAndValidate[DocumentPatchRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.PatchDocument(c.UserContext(), facetIdentity, documentIdentity, request.Input(), expected)
	return writeFacetDocument(c, fiber.StatusOK, value, err)
}

// DeleteDocument godoc
// @Summary Мягко удалить документ фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param documentIdentity path string true "Identity документа внутри фасета"
// @Param If-Match header string true "Текущая revision документа"
// @Success 200 {object} FacetDocumentResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents/{documentIdentity} [delete]
func (h *Handler) DeleteDocument(c *fiber.Ctx) error {
	return h.mutateDocument(c, h.usecase.DeleteDocument)
}

// RestoreDocument godoc
// @Summary Восстановить документ фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param documentIdentity path string true "Identity документа внутри фасета"
// @Param If-Match header string true "Текущая revision документа"
// @Success 200 {object} FacetDocumentResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents/{documentIdentity}/restore [post]
func (h *Handler) RestoreDocument(c *fiber.Ctx) error {
	return h.mutateDocument(c, h.usecase.RestoreDocument)
}

// ListRevisions godoc
// @Summary Получить ревизии фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Success 200 {object} revision.ListResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/revisions [get]
func (h *Handler) ListRevisions(c *fiber.Ctx) error {
	identity, err := path(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	values, err := h.usecase.ListRevisions(c.UserContext(), identity)
	return writeRevisions(c, values, err)
}

// GetRevision godoc
// @Summary Получить ревизию фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param revisionId path string true "UUID ревизии"
// @Success 200 {object} revision.Response
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/revisions/{revisionId} [get]
func (h *Handler) GetRevision(c *fiber.Ctx) error {
	identity, err := path(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.GetRevision(c.UserContext(), identity, c.Params("revisionId"))
	return writeRevision(c, value, err)
}

// RestoreRevision godoc
// @Summary Восстановить ревизию фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param revisionId path string true "UUID ревизии"
// @Param If-Match header string true "Текущая revision фасета"
// @Success 200 {object} FacetResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/revisions/{revisionId}/restore [post]
func (h *Handler) RestoreRevision(c *fiber.Ctx) error {
	identity, expected, err := identityAndRevision(c, "facetIdentity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.RestoreRevision(c.UserContext(), identity, c.Params("revisionId"), expected)
	return writeFacet(c, fiber.StatusOK, value, err)
}

// ListDocumentRevisions godoc
// @Summary Получить ревизии документа фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param documentIdentity path string true "Identity документа внутри фасета"
// @Success 200 {object} revision.ListResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents/{documentIdentity}/revisions [get]
func (h *Handler) ListDocumentRevisions(c *fiber.Ctx) error {
	facetIdentity, documentIdentity, err := documentPath(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	values, err := h.usecase.ListDocumentRevisions(c.UserContext(), facetIdentity, documentIdentity)
	return writeRevisions(c, values, err)
}

// GetDocumentRevision godoc
// @Summary Получить ревизию документа фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param documentIdentity path string true "Identity документа внутри фасета"
// @Param revisionId path string true "UUID ревизии"
// @Success 200 {object} revision.Response
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents/{documentIdentity}/revisions/{revisionId} [get]
func (h *Handler) GetDocumentRevision(c *fiber.Ctx) error {
	facetIdentity, documentIdentity, err := documentPath(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.GetDocumentRevision(c.UserContext(), facetIdentity, documentIdentity, c.Params("revisionId"))
	return writeRevision(c, value, err)
}

// RestoreDocumentRevision godoc
// @Summary Восстановить ревизию документа фасета
// @Tags Фасеты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param facetIdentity path string true "Identity фасета"
// @Param documentIdentity path string true "Identity документа внутри фасета"
// @Param revisionId path string true "UUID ревизии"
// @Param If-Match header string true "Текущая revision документа"
// @Success 200 {object} FacetDocumentResponse
// @Security BearerAuth
// @Router /api/v1/facets/{facetIdentity}/documents/{documentIdentity}/revisions/{revisionId}/restore [post]
func (h *Handler) RestoreDocumentRevision(c *fiber.Ctx) error {
	facetIdentity, documentIdentity, err := documentPath(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	expected, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.RestoreDocumentRevision(c.UserContext(), facetIdentity, documentIdentity, c.Params("revisionId"), expected)
	return writeFacetDocument(c, fiber.StatusOK, value, err)
}

func writeRevisions(c *fiber.Ctx, values []entities.Revision, err error) error {
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := shared.MapValues(values, revisionapi.NewResponse)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

func writeRevision(c *fiber.Ctx, value *entities.Revision, err error) error {
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := revisionapi.NewResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(response)
}

func (h *Handler) mutateDocument(c *fiber.Ctx, operation func(context.Context, string, string, int) (*entities.Document, error)) error {
	facetIdentity, documentIdentity, err := documentPath(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	expected, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := operation(c.UserContext(), facetIdentity, documentIdentity, expected)
	return writeFacetDocument(c, fiber.StatusOK, value, err)
}

func writeFacet(c *fiber.Ctx, status int, value *entities.Document, err error) error {
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := NewFacetResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.Status(status).JSON(response)
}

func writeFacetDocument(c *fiber.Ctx, status int, value *entities.Document, err error) error {
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := NewFacetDocumentResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.Status(status).JSON(response)
}

func path(c *fiber.Ctx, name string) (string, error) { return shared.PathParam(c, name) }

func identityAndRevision(c *fiber.Ctx, name string) (string, int, error) {
	identity, err := path(c, name)
	if err != nil {
		return "", 0, err
	}
	expected, err := shared.IfMatch(c)
	return identity, expected, err
}

func documentPath(c *fiber.Ctx) (string, string, error) {
	facetIdentity, err := path(c, "facetIdentity")
	if err != nil {
		return "", "", err
	}
	documentIdentity, err := path(c, "documentIdentity")
	return facetIdentity, documentIdentity, err
}
