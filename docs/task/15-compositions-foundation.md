# 15. Compositions Foundation

## Цель

Реализовать `RComposition` — source-first runtime graph without layout/rendering. Backend хранит canonical source и не сохраняет compiled graph, AST, diagnostics или runtime state.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля

```text
id UUID, workspace_id UUID, folder_id UUID,
identity TEXT, display_name TEXT, description TEXT NULL,
kind TEXT NOT NULL DEFAULT 'library', kind_identity TEXT NULL,
source TEXT, source_version INTEGER DEFAULT 1,
meta JSONB, active BOOLEAN, deleted_at TIMESTAMPTZ,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Allowed `kind` values:

```text
library | query | workspace | tenant | project | environment
```

`kind` определяет presentation ownership, но не меняет source/runtime semantics. `kindIdentity` содержит stable identity конкретного owner и остаётся portable text, а не polymorphic UUID foreign key.

- default: `kind=library`, `kindIdentity=null`;
- `library` располагается в обычном composition folder tree;
- contextual kind без `kindIdentity` располагается в корне соответствующей domain section;
- contextual kind с `kindIdentity` визуально прикрепляется к matching entity, а persisted folder не участвует в presentation routing;
- `project` всегда требует непустой `kindIdentity`;
- backend валидирует allowed kind и shape, а owner existence/dependency validation выполняется workspace-scoped rules задачи №7.

Uniqueness: `(workspace_id, identity)`; `sourceVersion >= 1`. Folder должна принадлежать текущему workspace. Composition не содержит `project_id`. Добавить folder type `compositions` и root `root-compositions`.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/compositions?kind=project&kind_identity=demo-project&folder_identity=...&include_deleted=false` — summaries без source; filters optional.
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
  "kind": "project",
  "kindIdentity": "demo-project",
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

PATCH может менять `displayName`, `description`, `kind`, `kindIdentity`, `folderIdentity`, `meta` и `active`. При смене `kind` пара `kind/kindIdentity` валидируется атомарно.

Errors: `composition_not_found`, `composition_identity_conflict`, `composition_in_use`, `invalid_composition_kind`, `composition_owner_required`, `composition_owner_not_found`, folder errors.

## Acceptance Criteria

List/detail разделены; source stored verbatim; `kind/kindIdentity` round-trip без зависимости от legacy metadata; runtime-derived fields отсутствуют в DB/API; lifecycle/OpenAPI/tests готовы; `go test ./...` проходит.
