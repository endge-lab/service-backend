# 06. Tenants Foundation

## Цель

Реализовать `RTenant` как workspace-scoped сущность: migration, entity, repository, usecase и HTTP API.

Задача зависит от `04-workspaces-foundation` и `05-workspace-context-middleware`. Backend открытый: users, memberships, roles и access checks не нужны. Все tenant endpoints используют workspace из `X-Endge-Workspace`.

## Поля модели

| API field | Database field | Обязательное | Описание |
|---|---|---:|---|
| `id` | `id UUID` | server only | Технический UUID tenant. |
| — | `workspace_id UUID` | да | Workspace-владелец из request context; не принимается из body. |
| `identity` | `identity TEXT` | да | Стабильный технический идентификатор tenant. |
| `displayName` | `display_name TEXT` | да | Отображаемое название. |
| `code` | `code TEXT` | да | Короткий бизнес-код tenant. |
| `description` | `description TEXT` | нет | Описание или `null`. |
| `folderIdentity` | `folder_id UUID` | нет | Identity папки типа `tenants`; по умолчанию `root-tenants`. |
| `createdAt` | `created_at TIMESTAMPTZ` | server only | Время создания. |
| `updatedAt` | `updated_at TIMESTAMPTZ` | server only | Время последнего изменения. |

Frontend-модель также содержит `name`, но это alias для `displayName`. Отдельное поле или колонку `name` создавать не нужно. Поля `meta`, `isSystem`, `active` и `deletedAt` в контракт `RTenant` не входят.

## Таблица `tenants`

Обновить `000003_init_tenants.sql`:

```sql
CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  identity TEXT NOT NULL,
  display_name TEXT NOT NULL,
  code TEXT NOT NULL,
  description TEXT NULL,
  folder_id UUID NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, identity),
  UNIQUE (workspace_id, code),
  CHECK (btrim(identity) <> ''),
  CHECK (btrim(display_name) <> ''),
  CHECK (btrim(code) <> '')
);
```

`identity` и `code` уникальны только внутри workspace. Один и тот же tenant identity может существовать в разных workspaces.

## Связь с папками

- в `000006_init_folders.sql` добавить `folders.workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE`;
- добавить `tenants` в допустимые `folders.entity_type`;
- для folders без проекта uniqueness должна учитывать workspace: `(workspace_id, entity_type, identity)`;
- уникальность root folder без проекта: `(workspace_id, entity_type) WHERE project_id IS NULL AND is_root`;
- `tenants.folder_id` связать с `folders.id` через foreign key с `ON DELETE SET NULL`;
- для каждого workspace должна существовать системная root folder:

```text
identity: root-tenants
entityType: tenants
projectId: null
isRoot: true
isSystem: true
```

- при создании tenant без `folderIdentity` использовать `root-tenants` текущего workspace;
- указанная папка должна принадлежать тому же workspace и иметь `entity_type = tenants`;
- HTTP API работает с `folderIdentity`, repository хранит разрешённый `folder_id` UUID;
- raw `folderId` из request body не принимать.

Provisioning `root-tenants` должен работать и для новых, и для уже существующих workspaces. Повторный запуск не должен создавать дубликаты.

## Entity, repository и usecase

Создать entity `RTenant` и port `TenantsRepository`.

Минимальные operations:

```text
Create(workspaceID, input) -> tenant
List(workspaceID, filter) -> tenants
GetByIdentity(workspaceID, identity) -> tenant
Update(workspaceID, identity, patch) -> tenant
HardDelete(workspaceID, identity) -> void
```

`workspaceID` берётся только из `WorkspaceFromContext`. Handler не читает header повторно, а usecase/repository не принимают workspace из body.

Update является partial update. Поля `id`, `workspaceID`, `identity`, `createdAt` и `updatedAt` неизменяемы. Изменять можно `displayName`, `code`, `description` и `folderIdentity`.

Удаление физическое: soft-delete и restore для `RTenant` не нужны. Если на tenant появятся database references, удаление занятого tenant должно возвращать conflict, а не удалять связанные данные неявно.

## HTTP API

Все методы требуют header:

```http
X-Endge-Workspace: default
```

### `GET /api/v1/tenants`

Возвращает все tenants активного workspace. Можно отфильтровать список по папке.

Query parameters:

```text
folder_identity = root-tenants  // optional
```

Пример запроса:

```http
GET /api/v1/tenants?folder_identity=root-tenants
X-Endge-Workspace: default
```

Response `200 OK`:

```json
{
  "items": [
    {
      "id": "00000000-0000-4000-8000-000000000201",
      "identity": "tenant-default",
      "displayName": "Default tenant",
      "code": "TENANT_DEFAULT",
      "description": "Main business tenant",
      "folderIdentity": "root-tenants",
      "createdAt": "2026-07-16T10:00:00Z",
      "updatedAt": "2026-07-16T10:00:00Z"
    }
  ]
}
```

### `POST /api/v1/tenants`

Создаёт tenant в workspace из header. `identity` и `code` должны быть уникальны внутри этого workspace.

Request body:

```json
{
  "identity": "tenant-default",
  "displayName": "Default tenant",
  "code": "TENANT_DEFAULT",
  "description": "Main business tenant",
  "folderIdentity": "root-tenants"
}
```

`folderIdentity` и `description` optional. Если папка не указана, используется `root-tenants`; если описание не указано, сохраняется `null`.

Response `201 Created`: созданный tenant в формате элемента list response. Duplicate `identity` или `code` возвращает `409 Conflict`.

### `GET /api/v1/tenants/:tenant_identity`

Возвращает один tenant активного workspace по `identity`.

Path parameter:

```text
tenant_identity = tenant-default
```

Пример запроса:

```http
GET /api/v1/tenants/tenant-default
X-Endge-Workspace: default
```

Response `200 OK`: tenant в формате элемента list response. Если tenant не найден в текущем workspace — `404 Not Found`.

### `PATCH /api/v1/tenants/:tenant_identity`

Частично обновляет tenant активного workspace.

Path parameter:

```text
tenant_identity = tenant-default
```

Request body example:

```json
{
  "displayName": "Main tenant",
  "code": "MAIN",
  "description": null,
  "folderIdentity": "root-tenants"
}
```

Response `200 OK`: обновлённый tenant. Неизвестная папка или папка другого workspace/type возвращает `400 Bad Request`; занятый `code` — `409 Conflict`; отсутствующий tenant — `404 Not Found`.

### `DELETE /api/v1/tenants/:tenant_identity`

Физически удаляет tenant из активного workspace.

Пример запроса:

```http
DELETE /api/v1/tenants/tenant-default
X-Endge-Workspace: default
```

Response `204 No Content`. Если tenant не найден — `404 Not Found`; если tenant используется другими записями — `409 Conflict`.

## Ошибки

Использовать общий error format сервиса:

```json
{
  "code": "validation_error",
  "message": "Validation error",
  "details": {}
}
```

Минимальные codes:

```text
validation_error
tenant_not_found
tenant_identity_conflict
tenant_code_conflict
tenant_in_use
folder_not_found
folder_workspace_mismatch
folder_entity_type_mismatch
internal_error
```

Ошибки workspace header (`workspace_required`, `workspace_not_found`) формирует middleware из задачи №5.

## Acceptance Criteria

- tenant всегда создаётся и читается только внутри `WorkspaceScope.ID`;
- `identity` и `code` изолированы по workspace;
- root folder `root-tenants` создаётся идемпотентно для каждого workspace;
- folder relation проверяется по workspace и `entity_type`;
- API реализован по examples выше и описан в OpenAPI;
- есть migration/repository/usecase/HTTP tests;
- проходит `go test ./...`.
