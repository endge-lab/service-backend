# 10. Types Foundation

## Цель

Реализовать `RType` как source-first описание доменного типа. Type принадлежит workspace и не привязан к project.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля и таблица

```text
id UUID, workspace_id UUID, identity TEXT, display_name TEXT,
source TEXT, source_version INTEGER, folder_id UUID, active BOOLEAN, is_system BOOLEAN,
is_primitive BOOLEAN, deleted_at TIMESTAMPTZ,
meta JSONB, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Обязательные defaults: `source=''`, `sourceVersion=1`, `active=true`, `isSystem=false`, `isPrimitive=false`, `meta={}`. Уникальность: `(workspace_id, identity)`. `identity` и `displayName` не пустые. Для пользовательского типа `source` не может быть пустым.

Canonical Type Source:

```ts
defineType({
  registration: field(String),
})
```

Type references внутри Source остаются строковыми identities и не заменяются UUID. Backend компилирует Source, валидирует references в текущем workspace и обновляет derived dependency index из задачи №7; hard-delete referenced type блокируется. При `isPrimitive=true` backend устанавливает `isSystem=true`; system type нельзя менять или удалять.

Добавить folder type `types` и root `root-types` для каждого workspace.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/types?folder_identity=root-types&include_deleted=false` — list; filters optional.
- `POST /api/v1/types` — создать type.
- `GET /api/v1/types/:type_ref` — получить по UUID или identity.
- `PATCH /api/v1/types/:type_id` — обновить по UUID.
- `DELETE /api/v1/types/:type_id` — soft-delete.
- `POST /api/v1/types/:type_id/restore` — восстановить.
- `DELETE /api/v1/types/:type_id/hard` — физически удалить, если type не используется.

Create request:

```json
{
  "identity": "Aircraft",
  "displayName": "Aircraft",
  "source": "defineType({ registration: field(String) })",
  "sourceVersion": 1,
  "folderIdentity": "root-types",
  "active": true,
  "isPrimitive": false,
  "meta": {}
}
```

PATCH принимает `displayName`, `source`, `sourceVersion`, `folderIdentity`, `active`, `meta`; `identity`, `id`, `isSystem`, timestamps immutable. Response возвращает request fields плюс UUID, deletion state и timestamps.

Errors: `type_not_found`, `type_identity_conflict`, `type_in_use`, `type_reference_not_found`, `system_type_mutation_forbidden`, folder errors.

## Acceptance Criteria

Type Source и type references компилируются, валидируются и индексируются без замены identities на UUID; hard-delete referenced type возвращает `type_in_use`; list не возвращает deleted types по умолчанию; все queries workspace-scoped; OpenAPI/tests готовы; `go test ./...` проходит.
