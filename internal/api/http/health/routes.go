// Package health exposes unversioned service health and version endpoints.
package health

import "github.com/gofiber/fiber/v2"

type Config struct {
	Service string
	Version string
	Env     string
}

func RegisterRoutes(router fiber.Router, cfg Config) {
	router.Get("/", healthHandler(cfg))
	router.Get("/health", healthHandler(cfg))
	router.Get("/version", versionHandler(cfg))
}

// healthHandler возвращает состояние работоспособности сервиса.
// @Summary Проверить работоспособность сервиса
// @Description Публичный liveness endpoint с именем сервиса, версией и окружением запуска.
// @ID getHealth
// @Tags Сервис
// @Produce json
// @Success 200 {object} HealthResponse "Сервис работает"
// @Router /health [get]
func healthHandler(cfg Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(HealthResponse{
			Status:  "ok",
			Service: cfg.Service,
			Version: cfg.Version,
			Env:     cfg.Env,
		})
	}
}

// versionHandler возвращает сведения о версии сервиса.
// @Summary Получить версию сервиса
// @Description Публичный endpoint с именем сервиса, версией сборки и окружением запуска.
// @ID getVersion
// @Tags Сервис
// @Produce json
// @Success 200 {object} VersionResponse "Версия сервиса"
// @Router /version [get]
func versionHandler(cfg Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(VersionResponse{
			Service: cfg.Service,
			Version: cfg.Version,
			Env:     cfg.Env,
		})
	}
}
