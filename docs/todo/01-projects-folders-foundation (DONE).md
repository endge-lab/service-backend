# 01. Projects + Folders Foundation

## Исходные условия

Считать, что в сервисе есть только базовый Go backend template: HTTP, config, DI, logger, postgres connection, migrations runner, health endpoint и общий формат ошибок.

Не опираться на старые миграции, старые таблицы или старую модель данных. Если в репозитории уже есть миграции для конфигуратора, их нужно считать черновиком прошлой модели, а не источником требований.

Эта задача должна создать первый минимальный бизнес-слой для конфигуратора: проекты и папки.

## Главное правило идентификаторов

Все UUID `id` генерируются на стороне backend.

Клиент явно передает только человекочитаемые `identity`:

- `projectIdentity` для ссылки на проект;
- `folderIdentity` для ссылки на папку;
- `parentIdentity` для ссылки на родительскую папку.

UUID можно возвращать в response как технический internal id, но публичный API не должен требовать от пользователя вручную передавать UUID при создании или связывании сущностей.

## Контекст

Проект - это изолированный контекст конфигурации. Все основные сущности конфигуратора в следующих задачах будут относиться к проекту.

Папка - это элемент дерева навигации внутри проекта и типа сущности. Например, у проекта может быть отдельное дерево папок для компонентов, converters, queries и data views.

Папки нужны сразу, потому что следующие сущности должны создаваться не "в воздухе", а внутри проекта и root-папки своего типа.

## Требуемые entity types

На этом этапе поддержать только entity types, которые нужны ближайшим задачам:

```text
components
converters
queries
data-views
```

Остальные типы не добавлять, пока под них нет API.

## Модель данных

Нужно создать новые миграции для таблиц `projects` и `folders`.

В БД связи хранятся через UUID foreign keys. В HTTP API эти UUID должны резолвиться по `identity`.

### projects

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL,
display_name TEXT NOT NULL,
description TEXT NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
deleted_at TIMESTAMPTZ NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Ограничения:

- `identity` уникален среди всех проектов.
- `identity` не пустой после trim.
- `display_name` не пустой после trim.
- `created_at` нельзя менять через API.
- `updated_at` обновляется при каждом update/soft-delete/restore.

### folders

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
project_id UUID NULL REFERENCES projects(id) ON DELETE CASCADE,
entity_type TEXT NOT NULL,
identity TEXT NOT NULL,
display_name TEXT NOT NULL,
description TEXT NULL,
parent_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
is_root BOOLEAN NOT NULL DEFAULT FALSE,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Ограничения:

- `entity_type` должен быть одним из поддержанных значений.
- `identity` уникален внутри пары `project_id + entity_type`.
- Для global/system folders с `project_id IS NULL` уникальность должна работать отдельно.
- В одном проекте и одном `entity_type` должна быть ровно одна root-folder.
- `parent_id` может ссылаться только на папку того же проекта и того же `entity_type`.
- Папка не может быть дочерней самой себе.
- Папка не может быть дочерней своему потомку.
- Root-папку нельзя переместить.
- System root-папку нельзя hard-delete.

## Поведение

При создании проекта сервис должен в одной транзакции:

1. Создать запись в `projects`.
2. Создать root-папки для entity types:
   - `components`
   - `converters`
   - `queries`
   - `data-views`

Формат root-folder identity:

```text
root-<entity_type>
```

Например:

```text
root-components-legacy
root-converters
root-queries
root-data-views
```

Root-папки создаются с:

```text
is_root = true
is_system = true
parent_id = null
```

Обычные list endpoints не должны возвращать записи с `deleted_at IS NOT NULL`.

Soft-delete должен только заполнить `deleted_at`.

Restore должен очистить `deleted_at`.

Hard-delete должен физически удалить запись, но запрещен для system root folders.

## API

Все endpoints должны быть под `/api/v1`.

### Projects

```text
GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/:project_identity
PATCH  /api/v1/projects/:project_identity
DELETE /api/v1/projects/:project_identity
POST   /api/v1/projects/:project_identity/restore
DELETE /api/v1/projects/:project_identity/hard
```

Create request:

```json
{
  "identity": "demo-project",
  "displayName": "Demo Project",
  "description": "Project for local configuration",
  "active": true,
  "meta": {}
}
```

Update request:

```json
{
  "displayName": "Demo Project",
  "description": "Updated description",
  "active": true,
  "meta": {}
}
```

Response:

```json
{
  "id": "00000000-0000-4000-8000-000000000001",
  "identity": "demo-project",
  "displayName": "Demo Project",
  "description": "Project for local configuration",
  "active": true,
  "deletedAt": null,
  "meta": {},
  "createdAt": "2026-07-08T10:00:00Z",
  "updatedAt": "2026-07-08T10:00:00Z"
}
```

### Folders

```text
GET    /api/v1/projects/:project_identity/folders?entity_type=components
POST   /api/v1/projects/:project_identity/folders
GET    /api/v1/projects/:project_identity/folders/:folder_identity?entity_type=components
PATCH  /api/v1/projects/:project_identity/folders/:folder_identity?entity_type=components
DELETE /api/v1/projects/:project_identity/folders/:folder_identity?entity_type=components
POST   /api/v1/projects/:project_identity/folders/:folder_identity/restore?entity_type=components
DELETE /api/v1/projects/:project_identity/folders/:folder_identity/hard?entity_type=components
```

Create request:

```json
{
  "entityType": "components",
  "identity": "shared-components",
  "displayName": "Shared Components",
  "description": "Reusable components",
  "parentIdentity": "root-components-legacy",
  "meta": {}
}
```

Response:

```json
{
  "id": "00000000-0000-4000-8000-000000000011",
  "projectIdentity": "demo-project",
  "entityType": "components",
  "identity": "shared-components",
  "displayName": "Shared Components",
  "description": "Reusable components",
  "parentIdentity": "root-components-legacy",
  "isRoot": false,
  "isSystem": false,
  "deletedAt": null,
  "meta": {},
  "createdAt": "2026-07-08T10:00:00Z",
  "updatedAt": "2026-07-08T10:00:00Z"
}
```

## Ошибки

Использовать единый JSON-формат:

```json
{
  "code": "validation_error",
  "message": "Validation error",
  "details": {}
}
```

Минимальные коды:

```text
validation_error
not_found
identity_conflict
folder_cycle
system_folder_delete_forbidden
internal_error
```

## Слои

Реализовать задачу через текущую clean architecture структуру:

- domain entities: `Project`, `Folder`;
- usecase ports: `ProjectsRepository`, `FoldersRepository`, `TxManager`;
- postgres repositories;
- HTTP handlers and transport DTO;
- bootstrap registration through `fx`;
- sqlc queries;
- migrations.

Usecase слой не должен импортировать postgres package.

## Tests

Минимально нужны:

- unit tests для валидации project/folder input;
- unit tests на запрет folder cycle;
- repository tests или интеграционные tests для unique identity;
- HTTP handler tests для create/list/get/update/delete/restore;
- `go test ./...` должен проходить.

## Acceptance Criteria

- Новые миграции создают только целевую модель из этого документа.
- `POST /api/v1/projects` создает проект и root folders в одной транзакции.
- `GET /api/v1/projects/demo-project/folders?entity_type=components` возвращает root-folder после создания проекта.
- Клиент не передает UUID при создании project/folder.
- Связи в публичном API задаются через identity.
- Soft-deleted проекты и папки не попадают в обычные списки.
- Нельзя создать дубликат `identity`.
- Нельзя создать цикл в дереве папок.
- Нельзя hard-delete system root folder.
- `go test ./...` проходит.
