# Migration TODO Report

Source of truth: `docs/endge-config-backend-task.md`.

## Not Created As Separate Tables

- `variables`: no standalone table definition in the document. Runtime variables are represented by `settings.vars JSONB NOT NULL DEFAULT '[]'::jsonb`.
- `auth`: no standalone table definition in the document. Auth configuration is represented by `settings.auth JSONB NULL` and query auth by `queries.auth JSONB NULL`.
- `sse`: no standalone table definition in the document. SSE configuration is represented by `settings.sse JSONB NULL`.
- `updates`: no standalone table definition in the document. Update configuration is represented by `settings.updates JSONB NOT NULL DEFAULT '[]'::jsonb`.
- `custom_sections`: no standalone table definition in the document. Custom sections are represented by `settings.custom_sections JSONB NOT NULL DEFAULT '[]'::jsonb`.
- `service_templates`: no table section, columns, types, or foreign keys are defined in the document.

## Existing Template Tables Outside The Document

- `todos`: already exists in `migrations/000002_create_todos.sql`, but no table section is defined in `docs/endge-config-backend-task.md`.
- `service_users`: already exists in `migrations/000001_init_service_template.sql`, but no table section is defined in `docs/endge-config-backend-task.md`.
