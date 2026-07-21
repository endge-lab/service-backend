# 16. Components Foundation

## Цель

Реализовать новый `RComponent` (`RComponentSFC`) как единственную развиваемую source-first component model. API resource: `/api/v1/components`; legacy collection в этой задаче не менять.

Задача зависит от `07-domain-relations-and-portable-import`.

## Persisted fields

```text
id UUID, workspace_id UUID, folder_id UUID,
identity TEXT, display_name TEXT, description TEXT NULL, tag TEXT NULL,
source TEXT, supported_targets JSONB, model_version INTEGER,
is_system BOOLEAN, meta JSONB, active BOOLEAN,
deleted_at TIMESTAMPTZ NULL, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Constants `kind=component-sfc`, `type=component-sfc`, `sourceKind=component-sfc` могут вычисляться transport/domain layer и не требуют отдельных колонок.

Rules:

- uniqueness `(workspace_id, identity)`;
- `identity` immutable и не может иметь UUID format;
- `supportedTargets` — непустой unique array из `dom | canvas`, default `['dom','canvas']`;
- `modelVersion >= 1`, default `1`;
- `tag` optional; duplicate/reserved tag не блокирует сохранение, а становится compiler diagnostic;
- folder принадлежит текущему workspace; component не содержит `project_id`;
- system component нельзя изменить или удалить через public API.

Добавить folder type `components` и root `root-components`.

Backend хранит canonical `source` verbatim. Не сохранять `sourceParts`, AST, IR, diagnostics, runtime dependencies, compiled artifacts или render state.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/components?identity=...&folder_identity=...&target=dom&include_deleted=false` — summaries без source.
- `POST /api/v1/components` — создать component.
- `GET /api/v1/components/:component_ref` — detail по UUID или identity.
- `PATCH /api/v1/components/:component_id` — metadata update только по UUID.
- `PUT /api/v1/components/:component_id/source` — полностью заменить canonical source.
- `POST /api/v1/components/:component_id/duplicate` — создать копию с новым identity; tag копии `null`.
- `DELETE /api/v1/components/:component_id` — soft-delete.
- `POST /api/v1/components/:component_id/restore` — restore.
- `DELETE /api/v1/components/:component_id/hard` — hard-delete, если component не используется.

Create request:

```json
{
  "identity": "user-card",
  "displayName": "User Card",
  "description": "Карточка пользователя",
  "tag": "UserCard",
  "folderIdentity": "root-components",
  "source": "<script>const props = defineProps({ name: String })</script>\n<template><div>{{ props.name }}</div></template>",
  "supportedTargets": ["dom"],
  "meta": {},
  "active": true
}
```

Source request:

```json
{
  "source": "<script>...</script>\n<template>...</template>\n<style>...</style>"
}
```

Duplicate request:

```json
{
  "identity": "user-card-copy",
  "displayName": "User Card Copy",
  "folderIdentity": "root-components"
}
```

List response содержит UUID, identity, metadata, targets, state and timestamps. Detail дополнительно содержит `source`. Backend должен разрешать сохранять incomplete/invalid source для editor autosave; compile/preview endpoints не нужны.

Errors: `component_not_found`, `component_identity_conflict`, `component_in_use`, `invalid_render_target`, `system_component_mutation_forbidden`, folder errors.

## Acceptance Criteria

Реализованы migration `000017`, entity/port/repository/usecase/HTTP/OpenAPI/tests; GET resolution deterministic (UUID → id, иначе identity); mutations UUID-only; `go test ./...` проходит.
