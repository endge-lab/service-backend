# 14. Computations Foundation

## Цель

Реализовать `RComputation` — persisted executable specification с canonical source и input/output contract. Backend не исполняет и не компилирует computation.

Задача зависит от `07-domain-relations-and-portable-import` и `10-types-foundation`.

## Поля

```text
id UUID, workspace_id UUID, folder_id UUID,
identity TEXT, display_name TEXT, description TEXT NULL, source TEXT,
source_version INTEGER, contract_version INTEGER,
input_type_id UUID NULL, input_is_array BOOLEAN NOT NULL DEFAULT FALSE,
input_optional BOOLEAN NOT NULL DEFAULT FALSE,
output_type_id UUID NULL, output_is_array BOOLEAN NOT NULL DEFAULT FALSE,
output_optional BOOLEAN NOT NULL DEFAULT FALSE,
meta JSONB, active BOOLEAN, deleted_at TIMESTAMPTZ,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Input/output contract:

```json
{
  "typeIdentity": "Orders",
  "isArray": true,
  "optional": false
}
```

`input` и `output` могут быть `null`; referenced type должен существовать в workspace. API принимает `typeIdentity`, usecase разрешает его в UUID, а repository сохраняет `input_type_id`/`output_type_id` и отдельные flags. JSON contract не является source of truth в БД.

Добавить composite foreign keys `(workspace_id, input_type_id)` и `(workspace_id, output_type_id)` на `types(workspace_id, id)`. Если contract равен `null`, соответствующий `*_type_id` равен `NULL`, а flags имеют `FALSE`. `sourceVersion >= 1`, `contractVersion >= 1`, `source` не пустой. Uniqueness: `(workspace_id, identity)`. Добавить folder type `computations` и root `root-computations`.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/computations?folder_identity=...&include_deleted=false` — summaries без source.
- `POST /api/v1/computations` — создать specification.
- `GET /api/v1/computations/:computation_ref` — detail по UUID или identity.
- `PATCH /api/v1/computations/:computation_id` — metadata/contract update по UUID.
- `PUT /api/v1/computations/:computation_id/source` — заменить canonical source.
- `DELETE /api/v1/computations/:computation_id` — soft-delete.
- `POST /api/v1/computations/:computation_id/restore` — restore.
- `DELETE /api/v1/computations/:computation_id/hard` — hard-delete, если computation не используется.

Create request:

```json
{
  "identity": "orders-total",
  "displayName": "Orders Total",
  "description": "Calculates order total",
  "folderIdentity": "root-computations",
  "source": "defineComputation(({ orders }) => orders.reduce((sum, x) => sum + x.total, 0))",
  "sourceVersion": 1,
  "contractVersion": 1,
  "input": { "typeIdentity": "Orders", "isArray": false, "optional": false },
  "output": { "typeIdentity": "number", "isArray": false, "optional": false },
  "meta": {},
  "active": true
}
```

PATCH changes metadata, relations, input/output, `contractVersion`, `active`; identity and `sourceVersion` immutable. Backend validates storage contract, not compiler diagnostics.

Transport mapper собирает `input`/`output` response обратно в публичный contract с `typeIdentity`; foreign UUID наружу не возвращается.

Computation не содержит `project_id`; контекстное использование формируется через source dependencies и typed bindings. Errors: `computation_not_found`, `computation_identity_conflict`, `computation_in_use`, `type_not_found`, `invalid_contract`, folder errors.

## Acceptance Criteria

Source and contract persisted separately from runtime artifacts; list excludes source; type identities из HTTP разрешаются в workspace-scoped UUID foreign keys; lifecycle/OpenAPI/tests готовы; `go test ./...` проходит.
