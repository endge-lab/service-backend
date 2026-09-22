package facet

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, handler *Handler) {
	resource := router.Group("/facets")
	resource.Post("/reorder", handler.Reorder)
	resource.Post("/", handler.Create)
	resource.Get("/", handler.List)
	resource.Post("/:facetIdentity/documents", handler.CreateDocument)
	resource.Get("/:facetIdentity/documents", handler.ListDocuments)
	resource.Get("/:facetIdentity/documents/:documentIdentity", handler.GetDocument)
	resource.Patch("/:facetIdentity/documents/:documentIdentity", handler.PatchDocument)
	resource.Delete("/:facetIdentity/documents/:documentIdentity", handler.DeleteDocument)
	resource.Post("/:facetIdentity/documents/:documentIdentity/restore", handler.RestoreDocument)
	resource.Get("/:facetIdentity/documents/:documentIdentity/revisions", handler.ListDocumentRevisions)
	resource.Get("/:facetIdentity/documents/:documentIdentity/revisions/:revisionId", handler.GetDocumentRevision)
	resource.Post("/:facetIdentity/documents/:documentIdentity/revisions/:revisionId/restore", handler.RestoreDocumentRevision)
	resource.Get("/:facetIdentity/revisions", handler.ListRevisions)
	resource.Get("/:facetIdentity/revisions/:revisionId", handler.GetRevision)
	resource.Post("/:facetIdentity/revisions/:revisionId/restore", handler.RestoreRevision)
	resource.Get("/:facetIdentity", handler.Get)
	resource.Patch("/:facetIdentity", handler.Patch)
	resource.Delete("/:facetIdentity", handler.Delete)
	resource.Post("/:facetIdentity/restore", handler.Restore)
}
