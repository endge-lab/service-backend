# 08. Environments Foundation

## Цель

Реализовать `REnvironment` — workspace-level окружение без привязки к project. Backend открытый; все endpoints используют `X-Endge-Workspace` и не проверяют пользователя.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля и таблица

```sql
CREATE TABLE environments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  identity TEXT NOT NULL,
  display_name TEXT NOT NULL,
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, identity),
  CHECK (btrim(identity) <> ''),
  CHECK (btrim(display_name) <> '')
);
```

API fields: `id`, `identity`, `displayName`, `isSystem`, `folderIdentity`, `createdAt`, `updatedAt`. Frontend field `name` — alias для `displayName`, отдельная колонка не нужна.

Добавить folder type `environments` и идемпотентный root `root-environments` в каждом workspace. Папка обязана принадлежать текущему workspace и иметь `entity_type = environments`.

## Usecase

```text
Create(workspaceID, input)
List(workspaceID, folderIdentity?)
GetByID / GetByIdentity
Update(workspaceID, id, patch)
HardDelete(workspaceID, id)
```

`identity` immutable. System environment нельзя менять или удалять через public API.

## HTTP API

```http
X-Endge-Workspace: default
```

- `GET /api/v1/environments?folder_identity=root-environments` — вернуть список окружений; query parameter optional.
- `POST /api/v1/environments` — создать окружение.
- `GET /api/v1/environments/:environment_ref` — получить по UUID или identity.
- `PATCH /api/v1/environments/:environment_id` — частично обновить по UUID.
- `DELETE /api/v1/environments/:environment_id` — физически удалить по UUID, response `204`.

Create request:

```json
{
  "identity": "production",
  "displayName": "Production",
  "folderIdentity": "root-environments"
}
```

Patch request:

```json
{
  "displayName": "Production EU",
  "folderIdentity": "root-environments"
}
```

Response `200/201`:

```json
{
  "id": "00000000-0000-4000-8000-000000000401",
  "identity": "production",
  "displayName": "Production",
  "isSystem": false,
  "folderIdentity": "root-environments",
  "createdAt": "2026-07-16T10:00:00Z",
  "updatedAt": "2026-07-16T10:00:00Z"
}
```

Errors: `environment_not_found`, `environment_identity_conflict`, `system_environment_mutation_forbidden`, `folder_not_found`, `folder_workspace_mismatch`, `folder_entity_type_mismatch`.

## Acceptance Criteria

Migration, entity, repository, usecase, HTTP/OpenAPI и tests реализованы; все queries содержат `workspace_id`; `go test ./...` проходит.
