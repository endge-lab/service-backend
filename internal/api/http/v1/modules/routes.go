package modules

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(router fiber.Router, h *Handler) {
	group := router.Group("/modules")
	group.Get("", h.list)
	group.Post("", h.create)
	group.Get("/:identity", h.detail)
	group.Patch("/:identity", h.patch)
	group.Delete("/:identity", h.softDelete)
	group.Post("/:identity/restore", h.restore)
	group.Post("/:identity/versions", h.publish)
	group.Get("/:identity/versions/:version", h.version)
	group.Get("/:identity/versions/:version/archive", h.archive)
	group.Get("/:identity/versions/:version/assets/:file", h.asset)
}
