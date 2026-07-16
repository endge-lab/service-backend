# 15. Compositions Foundation

## Цель

Реализовать `RComposition` — source-first runtime graph without layout/rendering. Backend хранит canonical source и не сохраняет compiled graph, AST, diagnostics или runtime state.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля

```text
id UUID, workspace_id UUID, project_id UUID NULL, folder_id UUID,
identity TEXT, display_name TEXT, description TEXT NULL,
source TEXT, source_version INTEGER DEFAULT 1,
meta JSONB, active BOOLEAN, inherited BOOLEAN, deleted_at TIMESTAMPTZ,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Uniqueness: `(workspace_id, identity)`; `sourceVersion >= 1`. Project/folder scope должен совпадать. Добавить folder type `compositions` и root `root-compositions`.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/compositions?project_identity=...&folder_identity=...&include_deleted=false` — summaries без source.
- `POST /api/v1/compositions` — создать composition.
- `GET /api/v1/compositions/:composition_ref` — detail по UUID или identity.
- `PATCH /api/v1/compositions/:composition_id` — metadata update по UUID.
- `PUT /api/v1/compositions/:composition_id/source` — заменить canonical source.
- `DELETE /api/v1/compositions/:composition_id` — soft-delete.
- `POST /api/v1/compositions/:composition_id/restore` — restore.
- `DELETE /api/v1/compositions/:composition_id/hard` — hard-delete, если нет references.

Create request:

```json
{
  "identity": "orders-page-runtime",
  "displayName": "Orders Page Runtime",
  "description": "Runtime graph for orders page",
  "projectIdentity": "demo-project",
  "folderIdentity": "root-compositions",
  "source": "defineComposition({ stores: ['orders-store'], nodes: [] })",
  "sourceVersion": 1,
  "meta": {},
  "active": true
}
```

Source update:

```json
{ "source": "defineComposition({ stores: ['orders-store'], nodes: [{ id: 'root' }] })" }
```

Backend разрешает сохранять work-in-progress source. Semantic references внутри source проверяет compiler, not persistence API.

Errors: `composition_not_found`, `composition_identity_conflict`, `composition_in_use`, project/folder errors.

## Acceptance Criteria

List/detail разделены; source stored verbatim; runtime-derived fields отсутствуют в DB/API; lifecycle/OpenAPI/tests готовы; `go test ./...` проходит.
