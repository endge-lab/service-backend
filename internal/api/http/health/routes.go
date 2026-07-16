// Package health exposes unversioned service health and version endpoints.
package health

import "github.com/gofiber/fiber/v2"

type Config struct {
	Service string
	Version string
	Env     string
}

func RegisterRoutes(app *fiber.App, cfg Config) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(HealthResponse{
			Status:  "ok",
			Service: cfg.Service,
			Version: cfg.Version,
			Env:     cfg.Env,
		})
	})
	app.Get("/version", func(c *fiber.Ctx) error {
		return c.JSON(VersionResponse{
			Service: cfg.Service,
			Version: cfg.Version,
			Env:     cfg.Env,
		})
	})
}
