# 03. Queries + Data Views

## Исходные условия

Считать, что в сервисе есть только базовый Go backend template и результаты задач:

1. `01-projects-folders-foundation`
2. `02-components-sfc-converters`

Не опираться на старые миграции, старые таблицы или старую модель `queries/views`, если она уже есть в репозитории. Целевая модель описана в этом документе.

## Главное правило идентификаторов

Все UUID `id` генерируются на стороне backend.

Клиент явно передает только человекочитаемые `identity`:

- `project_identity` в path;
- `folderIdentity` в request body;
- `query identity`;
- `queryIdentity` для связи DataView с Query;
- `dataView identity`.

Внутри БД связи хранятся через UUID foreign keys, но HTTP API должен резолвить их по identity. Клиент не должен вручную передавать `projectId`, `folderId` или `queryId`.

## Контекст

Query - это конфигурационный артефакт, который описывает источник данных. На первом этапе сервис должен хранить описание query, параметры, mock data и metadata. Сервис не должен выполнять query во внешнюю систему.

DataView - это конфигурационный артефакт, который описывает производное представление данных от query. DataView нужен, чтобы сохранить authoring model для преобразования результата query в форму, удобную UI-компонентам.

На этом этапе DataView должен быть самостоятельной backend-сущностью с явной связью на query. Не нужно прятать DataView внутрь frontend-only структуры и не нужно использовать старую таблицу `views`, если она была в прошлой модели.

## Модель данных

Нужно создать новые миграции для таблиц `queries` и `data_views`.

### queries

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE RESTRICT,
identity TEXT NOT NULL,
display_name TEXT NOT NULL,
description TEXT NULL,
query_type TEXT NOT NULL,
source JSONB NOT NULL DEFAULT '{}'::jsonb,
params JSONB NOT NULL DEFAULT '[]'::jsonb,
headers JSONB NOT NULL DEFAULT '{}'::jsonb,
auth JSONB NULL,
timeout_ms INTEGER NULL,
mock_data JSONB NULL,
mock_data_enabled BOOLEAN NOT NULL DEFAULT FALSE,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
active BOOLEAN NOT NULL DEFAULT TRUE,
deleted_at TIMESTAMPTZ NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Ограничения:

- `identity` уникален внутри проекта.
- `identity` не пустой после trim.
- `display_name` не пустой после trim.
- `query_type` не пустой после trim.
- `source` должен быть JSON object.
- `params` должен быть JSON array.
- `headers` должен быть JSON object.
- `timeout_ms` должен быть `NULL` или положительным числом.
- `folder_id` должен ссылаться на папку того же `project_id`.
- `folder_id` должен ссылаться на папку с `entity_type = queries`.

Примеры допустимых `query_type` на первом этапе:

```text
http
graphql
mock
```

Сервис должен хранить `query_type`, но не обязан исполнять ни один из этих типов.

### data_views

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE RESTRICT,
query_id UUID NOT NULL REFERENCES queries(id) ON DELETE CASCADE,
identity TEXT NOT NULL,
display_name TEXT NOT NULL,
description TEXT NULL,
view_type TEXT NOT NULL,
source JSONB NOT NULL DEFAULT '{}'::jsonb,
input_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
output_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
active BOOLEAN NOT NULL DEFAULT TRUE,
deleted_at TIMESTAMPTZ NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Ограничения:

- `identity` уникален внутри проекта.
- `identity` не пустой после trim.
- `display_name` не пустой после trim.
- `view_type` не пустой после trim.
- `source` должен быть JSON object.
- `input_schema` должен быть JSON object.
- `output_schema` должен быть JSON object.
- `query_id` должен ссылаться на query того же `project_id`.
- `folder_id` должен ссылаться на папку того же `project_id`.
- `folder_id` должен ссылаться на папку с `entity_type = data-views`.

Примеры допустимых `view_type` на первом этапе:

```text
pipeline
mapping
manual
```

Сервис должен хранить DataView authoring source, но не должен компилировать или исполнять его.

## API

Все endpoints должны быть под `/api/v1`.

### Queries

```text
GET    /api/v1/projects/:project_identity/queries?folder_identity=...&query_type=http
POST   /api/v1/projects/:project_identity/queries
GET    /api/v1/projects/:project_identity/queries/:query_identity
PATCH  /api/v1/projects/:project_identity/queries/:query_identity
DELETE /api/v1/projects/:project_identity/queries/:query_identity
POST   /api/v1/projects/:project_identity/queries/:query_identity/restore
DELETE /api/v1/projects/:project_identity/queries/:query_identity/hard
```

Create request:

```json
{
  "folderIdentity": "root-queries",
  "identity": "users-list",
  "displayName": "Users List",
  "description": "Loads users for the table",
  "queryType": "http",
  "source": {
    "method": "GET",
    "url": "https://example.test/users"
  },
  "params": [],
  "headers": {},
  "auth": null,
  "timeoutMs": 5000,
  "mockData": {
    "items": []
  },
  "mockDataEnabled": true,
  "meta": {},
  "active": true
}
```

### Data Views

```text
GET    /api/v1/projects/:project_identity/data-views?folder_identity=...&query_identity=...
POST   /api/v1/projects/:project_identity/data-views
GET    /api/v1/projects/:project_identity/data-views/:data_view_identity
PATCH  /api/v1/projects/:project_identity/data-views/:data_view_identity
DELETE /api/v1/projects/:project_identity/data-views/:data_view_identity
POST   /api/v1/projects/:project_identity/data-views/:data_view_identity/restore
DELETE /api/v1/projects/:project_identity/data-views/:data_view_identity/hard
```

Create request:

```json
{
  "folderIdentity": "root-data-views",
  "queryIdentity": "users-list",
  "identity": "users-table-view",
  "displayName": "Users Table View",
  "description": "Maps users query output to table rows",
  "viewType": "pipeline",
  "source": {
    "steps": []
  },
  "inputSchema": {},
  "outputSchema": {},
  "meta": {},
  "active": true
}
```

## Поведение

- Обычные list/get endpoints не возвращают soft-deleted записи.
- `DELETE /api/v1/projects/:project_identity/queries/:query_identity` делает soft-delete.
- `DELETE /api/v1/projects/:project_identity/data-views/:data_view_identity` делает soft-delete.
- Hard-delete физически удаляет запись.
- Если query hard-deleted, связанные data views удаляются каскадно.
- Если query soft-deleted, data views физически не удаляются, но обычные query-based сценарии не должны использовать soft-deleted query.
- Update должен заменять весь editable payload сущности, кроме `id`, `identity`, `created_at`, `deleted_at`.
- Если указан `folderIdentity` не того проекта или не того `entity_type`, вернуть validation error.
- Если указан `queryIdentity` из другого проекта, вернуть validation error.

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
folder_entity_type_mismatch
query_project_mismatch
internal_error
```

## Слои

Реализовать задачу через текущую clean architecture структуру:

- domain entities: `Query`, `DataView`;
- usecase ports: `QueriesRepository`, `DataViewsRepository`;
- postgres repositories;
- HTTP handlers and transport DTO;
- bootstrap registration through `fx`;
- sqlc queries;
- migrations.

Usecase слой не должен импортировать postgres package.

## Tests

Минимально нужны:

- validation tests для create/update query;
- validation tests для create/update data view;
- tests на `identity` conflict внутри одного проекта;
- tests на проверку folder `entity_type`;
- tests на запрет data view с query из другого проекта;
- HTTP handler tests для create/list/get/update/delete/restore;
- `go test ./...` должен проходить.

## Acceptance Criteria

- Новые миграции создают целевую модель из этого документа.
- Можно создать query с JSON `source`, params, headers и mock data.
- Сервис не исполняет query.
- Можно создать data view, связанный с query того же проекта.
- Сервис не компилирует и не исполняет data view source.
- Клиент не передает UUID при создании query/data view.
- Связи в публичном API задаются через identity.
- Нельзя создать query в folder с `entity_type != queries`.
- Нельзя создать data view в folder с `entity_type != data-views`.
- Нельзя создать data view для query из другого проекта.
- Soft-deleted записи не попадают в обычные списки.
- `go test ./...` проходит.
