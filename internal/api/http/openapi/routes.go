// Package openapi exposes the unversioned OpenAPI specification and documentation UI endpoints.
package openapi

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(app *fiber.App) {
	app.Get("/swagger/openapi3.yaml", handleOpenAPISpec)
	app.Get("/swagger", handleSwaggerUI)
}

func handleOpenAPISpec(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/yaml; charset=utf-8")
	return c.Send(openAPI3YAML)
}

func handleSwaggerUI(c *fiber.Ctx) error {
	return c.Type("html").SendString(`<!doctype html>
<html lang="ru">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Endge Service Backend Scalar</title>
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <div id="api-reference"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.28.5"></script>
    <script>
      Scalar.createApiReference('#api-reference', {
        url: '/swagger/openapi3.yaml',
        theme: 'blue',
        layout: 'modern',
        showSidebar: true,
        persistAuth: true,
        defaultOpenAllTags: false,
        authentication: {
          // Scoped operations require both the user token and workspace header.
          preferredSecurityScheme: [['BearerAuth', 'WorkspaceAuth']],
          securitySchemes: {
            WorkspaceAuth: {
              name: 'X-Endge-Workspace',
              in: 'header',
              value: 'demo-workspace',
            },
          },
        },
      })
    </script>
  </body>
</html>`)
}
