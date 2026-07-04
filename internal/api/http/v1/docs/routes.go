package http

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

type Config struct {
	OpenAPIPath string
}

func RegisterRoutes(app *fiber.App, cfg Config) {
	app.Get("/swagger/openapi.yaml", func(c *fiber.Ctx) error {
		return handleOpenAPISpec(c, cfg)
	})
	app.Get("/swagger", handleSwaggerUI)
}

func handleOpenAPISpec(c *fiber.Ctx, cfg Config) error {
	payload, err := os.ReadFile(cfg.OpenAPIPath)
	if err != nil {
		return err
	}

	c.Set("Content-Type", "application/yaml; charset=utf-8")
	return c.Send(payload)
}

func handleSwaggerUI(c *fiber.Ctx) error {
	return c.Type("html").SendString(`<!doctype html>
<html lang="ru">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Endge Service Template Scalar</title>
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/swagger/openapi.yaml"
      data-configuration='{"theme":"blue","layout":"modern","showSidebar":true,"persistAuth":true,"defaultOpenAllTags":false}'></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.28.5"></script>
  </body>
</html>`)
}
