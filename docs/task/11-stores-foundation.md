# 11. Stores Foundation

## Цель

Реализовать `RStore` как persisted source-first document. Backend хранит canonical `source`, но не компилирует его и не сохраняет AST, diagnostics или runtime state.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля

```text
id UUID, workspace_id UUID, folder_id UUID,
identity TEXT, display_name TEXT, description TEXT NULL,
source TEXT, source_version INTEGER DEFAULT 1,
meta JSONB DEFAULT {}, active BOOLEAN DEFAULT true,
deleted_at TIMESTAMPTZ NULL,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Уникальность: `(workspace_id, identity)`. `sourceVersion >= 1`. Folder должна принадлежать текущему workspace. Store не содержит `project_id`: contextual usage задаётся композициями и другими typed bindings. Добавить folder type `stores` и root `root-stores`.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/stores?folder_identity=...&include_deleted=false` — list summaries без `source`.
- `POST /api/v1/stores` — создать Store с canonical source.
- `GET /api/v1/stores/:store_ref` — detail по UUID или identity, включая `source`.
- `PATCH /api/v1/stores/:store_id` — обновить metadata по UUID.
- `PUT /api/v1/stores/:store_id/source` — полностью заменить `source`.
- `DELETE /api/v1/stores/:store_id` — soft-delete.
- `POST /api/v1/stores/:store_id/restore` — restore.
- `DELETE /api/v1/stores/:store_id/hard` — hard-delete, если нет references.

Create request:

```json
{
  "identity": "orders-store",
  "displayName": "Orders Store",
  "description": "Runtime state for orders",
  "folderIdentity": "root-stores",
  "source": "defineStore({ orders: [] })",
  "sourceVersion": 1,
  "meta": {},
  "active": true
}
```

Source request:

```json
{ "source": "defineStore({ orders: [], selectedId: null })" }
```

Backend разрешает сохранять незавершённый source: compiler validation живёт вне persistence service. PATCH принимает `displayName`, `description`, relations, `meta`, `active`; identity и sourceVersion immutable в v1.

Errors: `store_not_found`, `store_identity_conflict`, `store_in_use`, folder mismatch errors.

## Acceptance Criteria

List не читает source; detail/source endpoint возвращают canonical text; derived/runtime данные не persisted; soft-delete lifecycle, OpenAPI/tests и `go test ./...` готовы.
