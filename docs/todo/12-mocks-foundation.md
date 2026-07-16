# 12. Mocks Foundation

## Цель

Реализовать `RMock` — mock document с inline JSON/text или ссылкой на code provider. Backend хранит definition, но не вызывает provider.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля

```text
id UUID, workspace_id UUID, project_id UUID NULL, folder_id UUID,
identity TEXT, display_name TEXT, description TEXT NULL,
content_source TEXT, content_type TEXT, source TEXT NULL, code_ref TEXT NULL,
meta JSONB, active BOOLEAN, inherited BOOLEAN, deleted_at TIMESTAMPTZ,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Allowed values:

```text
contentSource: document | code-provider
contentType: application/json | text/plain
```

Rules:

- uniqueness `(workspace_id, identity)`;
- `document`: `source` обязателен; при JSON он должен парситься;
- `code-provider`: требуется namespaced `codeRef`, например `@app:mocks.orders`; inline source не используется;
- maximum source size — 2 MB;
- project/folder должны принадлежать текущему workspace и совпадать между собой.

Добавить folder type `mocks` и root `root-mocks`.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/mocks?project_identity=...&folder_identity=...&content_source=document` — summaries без `source`.
- `POST /api/v1/mocks` — создать mock.
- `GET /api/v1/mocks/:mock_ref` — получить полный mock по UUID или identity.
- `PATCH /api/v1/mocks/:mock_id` — обновить metadata/config по UUID.
- `PUT /api/v1/mocks/:mock_id/source` — заменить inline source для `contentSource=document`.
- `DELETE /api/v1/mocks/:mock_id` — soft-delete.
- `POST /api/v1/mocks/:mock_id/restore` — restore.
- `DELETE /api/v1/mocks/:mock_id/hard` — hard-delete.

Create inline JSON:

```json
{
  "identity": "orders-list",
  "displayName": "Orders List",
  "projectIdentity": "demo-project",
  "folderIdentity": "root-mocks",
  "contentSource": "document",
  "contentType": "application/json",
  "source": "{\"items\":[]}",
  "meta": {}
}
```

Create code provider:

```json
{
  "identity": "orders-provider",
  "displayName": "Orders Provider",
  "folderIdentity": "root-mocks",
  "contentSource": "code-provider",
  "codeRef": "@app:mocks.orders"
}
```

PATCH может менять `displayName`, `description`, relations, content config, `meta`, `active`; identity immutable. Errors: `mock_not_found`, `mock_identity_conflict`, `invalid_mock_json`, `mock_source_too_large`, `code_ref_required`, folder/project errors.

## Acceptance Criteria

Conditional validation реализована в usecase; provider не исполняется; list не читает source; soft-delete/OpenAPI/tests готовы; `go test ./...` проходит.
