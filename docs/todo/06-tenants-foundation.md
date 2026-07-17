# 06. Tenants Foundation

## Цель

Реализовать `RTenant` как workspace-scoped entity и финальный configuration layer: migration, entity, repository, usecase и HTTP API.

Задача зависит от `04-workspaces-foundation` и `05-workspace-context-middleware`. Backend открытый: users, memberships, roles и access checks не нужны. Все tenant endpoints используют workspace из `X-Endge-Workspace`.

Tenant не владеет Project и не содержит список проектов. Он может участвовать в нескольких execution contexts внутри одного workspace. Конкретное сочетание Tenant + Project + Environment выбирается при boot.

## Поля модели

| API field | Database field | Обязательное | Описание |
|---|---|---:|---|
| `id` | `id UUID` | server only | Технический UUID tenant. |
| — | `workspace_id UUID` | да | Workspace из request context; не принимается из body. |
| `identity` | `identity TEXT` | да | Стабильный identity tenant. |
| `displayName` | `display_name TEXT` | да | Отображаемое название. |
| `code` | `code TEXT` | да | Короткий бизнес-код tenant. |
| `description` | `description TEXT` | нет | Описание или `null`. |
| `folderIdentity` | `folder_id UUID` | нет | Папка типа `tenants`; по умолчанию `root-tenants`. |
| `configuration` | `configuration JSONB` | да | `EndgeConfigurationContribution`. |
| `createdAt` | `created_at TIMESTAMPTZ` | server only | Время создания. |
| `updatedAt` | `updated_at TIMESTAMPTZ` | server only | Время изменения. |

Frontend field `name` является alias для `displayName`; отдельную колонку `name` создавать не нужно.

## Таблица `tenants`

```sql
CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  identity TEXT NOT NULL,
  display_name TEXT NOT NULL,
  code TEXT NOT NULL,
  description TEXT NULL,
  folder_id UUID NULL,
  configuration JSONB NOT NULL DEFAULT '{"mode":"inherit","patch":{}}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, identity),
  UNIQUE (workspace_id, code),
  CHECK (btrim(identity) <> ''),
  CHECK (btrim(display_name) <> ''),
  CHECK (btrim(code) <> ''),
  CHECK (jsonb_typeof(configuration) = 'object')
);
```

## Configuration contribution

Default означает чистое наследование:

```json
{
  "mode": "inherit",
  "patch": {}
}
```

В `inherit` сохраняются только локальные operations. Collection fields используют `upsert/remove`, scalar fields — `set/remove`:

```json
{
  "mode": "inherit",
  "patch": {
    "themes": {
      "entries": [
        {
          "key": "tenant-brand",
          "op": "upsert",
          "value": {
            "identity": "tenant-brand",
            "displayName": "Tenant Brand"
          }
        },
        { "key": "dark", "op": "remove" }
      ]
    },
    "defaultTheme": {
      "op": "set",
      "value": "tenant-brand"
    }
  }
}
```

`replace` полностью отбрасывает upstream configuration и содержит полный contract из задачи №4:

```json
{
  "mode": "replace",
  "value": {
    "vars": [],
    "locales": [
      { "code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr" }
    ],
    "defaultLocale": "ru",
    "fallbackLocale": "ru",
    "themes": [
      { "identity": "light", "displayName": "Светлая" }
    ],
    "defaultTheme": "light",
    "defaultAuthProfileIdentity": null,
    "sfcAdapterIds": ["native-vue"],
    "defaultSfcAdapterId": "native-vue"
  }
}
```

Допустимые patch keys:

```text
collections: vars, locales, themes, sfcAdapterIds
scalars: sse, defaultLocale, fallbackLocale, defaultTheme,
         defaultAuthProfileIdentity, defaultSfcAdapterId
```

Required scalar fields нельзя удалять. `remove` для `defaultAuthProfileIdentity` означает explicit `null`, а для `sse` — отсутствие SSE configuration.

Usecase валидирует contribution structurally. Для `replace` выполняется полная validation `EndgeConfiguration`. Effective configuration в tenant record не сохранять.

## Resolution semantics

Tenant применяется последним:

```text
Workspace -> Project -> Environment -> Tenant
```

Это означает, что tenant может делать небольшие branding/locale/auth overrides поверх выбранного Project и Environment. Нельзя заранее вычислить tenant upstream без полного execution context.

## Связь с папками

- добавить `tenants` в допустимые `folders.entity_type`;
- для каждого workspace создать system root `root-tenants`;
- при создании tenant без `folderIdentity` использовать `root-tenants` текущего workspace;
- folder должна принадлежать тому же workspace и иметь `entity_type = tenants`;
- HTTP API принимает `folderIdentity`, repository хранит UUID;
- raw `folderId` из request body не принимать.

Provisioning root folder должен быть идемпотентным.

## Entity, repository и usecase

Создать `RTenant` и `TenantsRepository`.

```text
Create(workspaceID, input) -> tenant
List(workspaceID, filter) -> tenants
GetByIdentity(workspaceID, identity) -> tenant
Update(workspaceID, identity, patch) -> tenant
HardDelete(workspaceID, identity) -> void
```

Update partial. Неизменяемы `id`, `workspaceID`, `identity`, `createdAt`, `updatedAt`. Изменяемы `displayName`, `code`, `description`, `folderIdentity`, `configuration`.

Если передано `configuration`, contribution заменяется целиком. Backend не выполняет дополнительный merge с предыдущим contribution: операции inherit уже являются явной persisted delta.

## HTTP API

Все методы требуют:

```http
X-Endge-Workspace: default
```

```text
GET    /api/v1/tenants?folder_identity=root-tenants
POST   /api/v1/tenants
GET    /api/v1/tenants/:tenant_identity
PATCH  /api/v1/tenants/:tenant_identity
DELETE /api/v1/tenants/:tenant_identity
```

Create request:

```json
{
  "identity": "tenant-default",
  "displayName": "Default tenant",
  "code": "TENANT_DEFAULT",
  "description": "Main business tenant",
  "folderIdentity": "root-tenants",
  "configuration": {
    "mode": "inherit",
    "patch": {}
  }
}
```

`configuration` optional on create; default is clean inherit.

Response:

```json
{
  "id": "00000000-0000-4000-8000-000000000201",
  "identity": "tenant-default",
  "displayName": "Default tenant",
  "code": "TENANT_DEFAULT",
  "description": "Main business tenant",
  "folderIdentity": "root-tenants",
  "configuration": {
    "mode": "inherit",
    "patch": {}
  },
  "createdAt": "2026-07-16T10:00:00Z",
  "updatedAt": "2026-07-16T10:00:00Z"
}
```

Удаление физическое. Если tenant используется другими persisted records, вернуть `409 tenant_in_use`. Отсутствие runtime context binding не является database relation и не блокирует delete.

## Errors

```text
validation_error
tenant_not_found
tenant_identity_conflict
tenant_code_conflict
tenant_in_use
folder_not_found
folder_workspace_mismatch
folder_entity_type_mismatch
configuration_invalid
internal_error
```

## Tests и Acceptance Criteria

- tenant всегда создаётся и читается внутри `WorkspaceScope.ID`;
- tenant не содержит `project_id` или `environment_id`;
- default configuration равен `{ mode: "inherit", patch: {} }`;
- repository сохраняет contribution без вычисления effective configuration;
- usecase валидирует `inherit`, `replace`, `upsert`, `remove` и запрещённое удаление required scalars;
- identity и code изолированы по workspace;
- root folder `root-tenants` создаётся идемпотентно;
- API и OpenAPI соответствуют examples;
- есть migration/repository/usecase/HTTP tests;
- проходит `go test ./...`.
