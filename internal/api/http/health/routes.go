// Package health exposes unversioned service health and version endpoints.
package health

import (
	"context"

	"github.com/endge-lab/service-backend/internal/usecase/service_info"
	"github.com/gofiber/fiber/v2"
)

type Config struct {
	Service string
	Version string
	Env     string
}

type ConnectedServices interface {
	List(context.Context) []service_info.ConnectedService
}

func RegisterRoutes(router fiber.Router, cfg Config, connectedServices ConnectedServices) {
	router.Get("/", healthHandler(cfg))
	router.Get("/health", healthHandler(cfg))
	router.Get("/version", versionHandler(cfg, connectedServices))
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
// @Description Публичный endpoint с версией backend и доступностью подключённых backend-сервисов.
// @ID getVersion
// @Tags Сервис
// @Produce json
// @Success 200 {object} VersionResponse "Версия сервиса"
// @Router /version [get]
func versionHandler(cfg Config, connectedServices ConnectedServices) fiber.Handler {
	return func(c *fiber.Ctx) error {
		services := connectedServices.List(c.UserContext())
		responseServices := make([]ConnectedServiceResponse, 0, len(services))
		for _, service := range services {
			responseServices = append(responseServices, ConnectedServiceResponse{
				Service: service.Service,
				Version: service.Version,
				Env:     service.Env,
				Status:  service.Status,
			})
		}
		return c.JSON(VersionResponse{
			Service:  cfg.Service,
			Version:  cfg.Version,
			Env:      cfg.Env,
			Services: responseServices,
		})
	}
}
