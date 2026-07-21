# 09. Versions Foundation

## Цель

Реализовать `RVersion` — immutable snapshot доменных данных. List загружает только metadata, а тяжёлое поле `data` возвращается только detail endpoint.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля и таблица

```sql
CREATE TABLE versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  identity TEXT NOT NULL,
  description TEXT NULL,
  data JSONB NOT NULL,
  project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, identity),
  CHECK (btrim(identity) <> '')
);
```

API fields: `id`, `identity`, `description`, `projectIdentity`, `data`, `createdAt`, `updatedAt`. `data` — любой валидный JSON domain dump. Project обязан принадлежать текущему workspace.

## Usecase

```text
Create(workspaceID, input)
List(workspaceID, projectIdentity?)
GetByID / GetByIdentity
HardDelete(workspaceID, id)
```

Version после создания не изменяется: PATCH, soft-delete и restore не нужны.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/versions?project_identity=demo-project` — список metadata без `data`; filter optional.
- `POST /api/v1/versions` — сохранить новый snapshot.
- `GET /api/v1/versions/:version_ref` — получить snapshot с `data` по UUID или identity.
- `DELETE /api/v1/versions/:version_id` — физически удалить snapshot по UUID, response `204`.

Create request:

```json
{
  "identity": "release-2026-07-16",
  "description": "Snapshot before release",
  "projectIdentity": "demo-project",
  "data": {
    "projects": [],
    "components": [],
    "queries": []
  }
}
```

List response `200`:

```json
{
  "items": [
    {
      "id": "00000000-0000-4000-8000-000000000402",
      "identity": "release-2026-07-16",
      "description": "Snapshot before release",
      "projectIdentity": "demo-project",
      "createdAt": "2026-07-16T10:00:00Z",
      "updatedAt": "2026-07-16T10:00:00Z"
    }
  ]
}
```

Detail response имеет те же поля и дополнительно `data`. Errors: `version_not_found`, `version_identity_conflict`, `project_not_found`, `project_workspace_mismatch`, `invalid_snapshot_data`.

## Acceptance Criteria

List SQL не выбирает `data`; snapshots immutable; scope берётся только из context; migration/repository/usecase/HTTP tests и `go test ./...` проходят.
