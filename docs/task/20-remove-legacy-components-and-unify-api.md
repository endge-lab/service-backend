# 20. Remove Legacy Components and Align Tasks 01–03

## Цель

Доработать только модели и API, которые уже были реализованы в задачах №1–3, и удалить временный legacy component stack из задачи №2.

Эта задача не изменяет новые модели из задач №4–19. Они должны сразу реализовываться по новому contract в своих собственных задачах.

Задача выполняется последней, потому что использует уже готовые:

- Workspace и `X-Endge-Workspace` из задач №4–5;
- relation resolver, dependency index и portable contracts из задачи №7;
- canonical `/api/v1/components` из задачи №16.

## Точный scope

Дорабатывать в этой задаче:

```text
№1: Projects
№1: Folders
№2: Components SFC, временно реализованные как components_legacy
№2: Converters
№3: Queries
№3: Data Views
```

Не дорабатывать и не перепроверять здесь:

```text
Workspaces
Tenants
Environments
Versions
Types
Stores
Mocks
Vocabs
Computations
Compositions
новые Components из задачи №16
Filters
Auth Profiles
Navigations
```

Для перечисленных новых моделей их migrations, entities, repositories, usecases, HTTP API, dependencies и portable adapters являются ответственностью соответствующих задач №4–19.

Разрешено записать migrated SFC records в уже готовую таблицу `components` через contract задачи №16. Это не означает изменение schema/API новой Component model.

## 1. Безопасный перенос и удаление `components_legacy`

`components_legacy` — это временное имя SFC-модели, созданной в задаче №2. Не считать её DSL/Table моделью и не пытаться применять converter для старых Payload component types.

Перед migration выполнить inventory:

```text
общее количество components_legacy
количество active и soft-deleted records
workspace и project для каждой записи
используемые folders
конфликты identity с canonical components
references по UUID или identity
```

### Mapping в canonical `components`

Каждую запись переносить в модель из задачи №16:

```text
workspace_id       ← projects.workspace_id
project_id         ← components_legacy.project_id
folder_id          ← новая components folder
identity           ← identity
display_name       ← display_name
description        ← description
source             ← source
tag                ← NULL
supported_targets  ← ["dom"]
model_version      ← 1
active             ← active
deleted_at         ← deleted_at
created_at         ← created_at
updated_at         ← updated_at
```

Folder mapping:

- `root-components-legacy` переносится в `root-components` текущего workspace/project;
- custom legacy component folders переносятся под `root-components` с сохранением identity, display name, tree и metadata;
- folder conflict нельзя разрешать silent overwrite: сформировать report и остановить migration.

Поля `props_schema` и `bindings` нельзя терять. Сохранить их в:

```json
{
  "meta": {
    "legacyImport": {
      "propsSchema": {},
      "bindings": {}
    }
  }
}
```

Это data-preserving migration, но не обещание runtime compatibility. Backend не должен преобразовывать эти поля в новый SFC source.

Migration rules:

1. Если `(workspace_id, identity)` уже занят в canonical `components`, остановить migration и добавить conflict в report.
2. Не перезаписывать существующий canonical component.
3. По возможности сохранить legacy UUID; если UUID уже занят, остановить migration, а не создавать скрытый remap.
4. Перенести soft-delete state и timestamps.
5. После записи сравнить количество source records и migrated records, identities и source checksum.
6. Удалять legacy stack только после полной успешной проверки.
7. Если найдена reference, которую нельзя безопасно remap, остановить migration и вернуть понятную ошибку.

После успешного переноса удалить:

```text
components_legacy table and indexes
/api/v1/projects/:project_identity/components-legacy routes
legacy OpenAPI operations
component_legacy handlers/transports/usecases/ports
component_legacy repositories/mappers/sqlc queries
RComponentLegacy backend entity
components-legacy folder entity type
root-components-legacy provisioning
legacy bootstrap registrations and tests
```

`/api/v1/components` из задачи №16 остаётся единственным component API. Его schema, handlers и public contract в №20 не переделывать.

## 2. Доработка Projects, Folders, Converters, Queries и Data Views

Применить новый contract только к оставшимся моделям из задач №1–3.

### Workspace scope

- все их endpoints требуют `X-Endge-Workspace: <workspace_identity>`;
- `workspaceId` и `workspaceIdentity` не принимаются из body/path/query;
- handlers получают `WorkspaceScope` из context задачи №5;
- usecases и repositories всегда получают verified `workspaceID`;
- данные другого workspace не возвращаются даже при обращении по UUID.

`projects.workspace_id` добавляется задачей №4. В №20 не повторять эту migration, а использовать готовую relation.

Добавить `workspace_id UUID NOT NULL` в оставшиеся таблицы из №1–3:

```text
folders
converters
queries
data_views
```

Backfill выполнять через `projects.workspace_id`. Если найдены legacy folders с `project_id IS NULL`, их workspace невозможно определить автоматически: сформировать report и остановить migration. Не назначать их в `default` workspace молча.

Затем добавить composite foreign keys, запрещающие cross-workspace и cross-project relations. Минимальный набор:

```text
folders(workspace_id, project_id)
  → projects(workspace_id, id)

folders(workspace_id, project_id, entity_type, parent_id)
  → folders(workspace_id, project_id, entity_type, id)

converters(workspace_id, project_id, folder_id)
  → folders(workspace_id, project_id, id)

queries(workspace_id, project_id, folder_id)
  → folders(workspace_id, project_id, id)

data_views(workspace_id, project_id, folder_id)
  → folders(workspace_id, project_id, id)

data_views(workspace_id, project_id, query_id)
  → queries(workspace_id, project_id, id)
```

Добавить соответствующие target `UNIQUE` constraints для composite FK. Не ограничиваться только application validation: database должна блокировать повреждённую relation при race condition или прямом SQL write.

### Uniqueness

```text
projects    → UNIQUE (workspace_id, identity)
folders     → UNIQUE (workspace_id, project_id, entity_type, identity)
converters  → UNIQUE (workspace_id, identity)
queries     → UNIQUE (workspace_id, identity)
data_views  → UNIQUE (workspace_id, identity)
```

Перед изменением constraints сформировать report для identities, которые раньше были допустимы в разных projects одного workspace. При конфликте migration должна остановиться; автоматически переименовывать записи запрещено.

### ID и identity

```text
POST create                  → identity в body
GET detail                   → UUID или identity
PATCH/PUT/DELETE/restore     → только UUID
Database relations           → UUID foreign keys
HTTP relation input/output   → ...Identity
```

GET resolver:

```text
валидный UUID → GetByID(workspaceID, id), без fallback
иначе         → GetByIdentity(workspaceID, identity)
```

Identity immutable и не может иметь UUID format. UUID возвращается как JSON string.

### Relations

Public DTO принимает:

```json
{
  "projectIdentity": "demo-project",
  "folderIdentity": "root-queries",
  "queryIdentity": "orders-query"
}
```

Usecase разрешает identities в UUID через task №7. `projectId`, `folderId`, `queryId` из public body не принимать.

Обязательные database checks:

- folder принадлежит тому же workspace/project и имеет ожидаемый `entity_type`;
- converter/query/data view принадлежат тому же workspace, что project и folder;
- `data_views.query_id` указывает на query того же workspace и project;
- query hard-delete блокируется, пока существуют data views;
- project/folder hard-delete блокируется, пока существуют child folders или documents;
- converter/query/data view hard-delete использует dependency guard задачи №7;
- не использовать `CASCADE` для удаления самостоятельных domain documents.

## 3. Целевой HTTP API для моделей №1–3

Все примеры ниже требуют:

```http
X-Endge-Workspace: default
```

### Projects

- `GET /api/v1/projects?include_deleted=false` — возвращает projects текущего workspace; query parameter optional.
- `POST /api/v1/projects` — создаёт project; body содержит `identity`, `displayName`, optional `description`, `active`, `meta`.
- `GET /api/v1/projects/:project_ref` — возвращает project по UUID или identity; пример `GET /api/v1/projects/demo-project`.
- `PATCH /api/v1/projects/:project_id` — частично обновляет metadata по UUID; пример body `{ "displayName": "Demo" }`.
- `DELETE /api/v1/projects/:project_id` — выполняет soft-delete project по UUID.
- `POST /api/v1/projects/:project_id/restore` — восстанавливает soft-deleted project по UUID; body отсутствует.
- `DELETE /api/v1/projects/:project_id/hard` — физически удаляет project только при отсутствии зависимостей.

### Folders

- `GET /api/v1/folders?project_identity=demo-project&entity_type=queries&parent_identity=root-queries&include_deleted=false` — возвращает folders по указанному project/type; filters optional, кроме `entity_type`.
- `POST /api/v1/folders` — создаёт folder; body содержит `projectIdentity`, `entityType`, `identity`, `displayName`, optional `parentIdentity`, `description`, `meta`.
- `GET /api/v1/folders/:folder_ref?project_identity=demo-project&entity_type=queries` — возвращает folder по UUID или identity. Для identity обязательно передать `project_identity` и `entity_type`.
- `PATCH /api/v1/folders/:folder_id` — частично обновляет metadata или `parentIdentity` по UUID.
- `DELETE /api/v1/folders/:folder_id` — выполняет soft-delete folder по UUID.
- `POST /api/v1/folders/:folder_id/restore` — восстанавливает folder по UUID; body отсутствует.
- `DELETE /api/v1/folders/:folder_id/hard` — физически удаляет folder при отсутствии children/documents; system root удалить нельзя.

### Converters

- `GET /api/v1/converters?project_identity=demo-project&folder_identity=root-converters&include_deleted=false` — возвращает converter summaries без `source`; filters optional.
- `POST /api/v1/converters` — создаёт converter; body содержит identity/metadata, `projectIdentity`, `folderIdentity`, `converterType` и `source`.
- `GET /api/v1/converters/:converter_ref` — возвращает полный converter по UUID или identity; пример `GET /api/v1/converters/date-to-display`.
- `PATCH /api/v1/converters/:converter_id` — частично обновляет metadata, relations и active state по UUID; `source` здесь не принимается.
- `PUT /api/v1/converters/:converter_id/source` — полностью заменяет canonical JSON `source`; body `{ "source": { "kind": "template", "template": "{{ value }}" } }`.
- `DELETE /api/v1/converters/:converter_id` — выполняет soft-delete converter по UUID.
- `POST /api/v1/converters/:converter_id/restore` — восстанавливает converter по UUID; body отсутствует.
- `DELETE /api/v1/converters/:converter_id/hard` — hard-delete при отсутствии usages; system converter удалить нельзя.

### Queries

- `GET /api/v1/queries?project_identity=demo-project&folder_identity=root-queries&query_type=http&include_deleted=false` — возвращает query summaries без тяжёлого authoring payload; filters optional.
- `POST /api/v1/queries` — создаёт query; body содержит identity/metadata, `projectIdentity`, `folderIdentity`, `queryType` и authoring fields из задачи №3.
- `GET /api/v1/queries/:query_ref` — возвращает полный query по UUID или identity; пример `GET /api/v1/queries/users-list`.
- `PATCH /api/v1/queries/:query_id` — частично обновляет metadata, relations и active state по UUID; authoring payload здесь не принимается.
- `PUT /api/v1/queries/:query_id/source` — полностью заменяет authoring fields `source`, `params`, `headers`, `auth`, `timeoutMs`, `mockData`, `mockDataEnabled` одним request.
- `DELETE /api/v1/queries/:query_id` — выполняет soft-delete query по UUID.
- `POST /api/v1/queries/:query_id/restore` — восстанавливает query по UUID; body отсутствует.
- `DELETE /api/v1/queries/:query_id/hard` — hard-delete только при отсутствии data views и dependency usages.

Query source request example:

```json
{
  "source": { "method": "GET", "url": "https://example.test/users" },
  "params": [],
  "headers": {},
  "auth": null,
  "timeoutMs": 5000,
  "mockData": { "items": [] },
  "mockDataEnabled": true
}
```

### Data Views

- `GET /api/v1/data-views?project_identity=demo-project&folder_identity=root-data-views&query_identity=users-list&include_deleted=false` — возвращает data view summaries без source/schemas; filters optional.
- `POST /api/v1/data-views` — создаёт data view; body содержит identity/metadata, `projectIdentity`, `folderIdentity`, `queryIdentity`, `viewType`, source и schemas.
- `GET /api/v1/data-views/:data_view_ref` — возвращает полный data view по UUID или identity; пример `GET /api/v1/data-views/users-table-view`.
- `PATCH /api/v1/data-views/:data_view_id` — частично обновляет metadata, relations и active state по UUID; authoring payload здесь не принимается.
- `PUT /api/v1/data-views/:data_view_id/source` — полностью заменяет `source`, `inputSchema` и `outputSchema` одним request.
- `DELETE /api/v1/data-views/:data_view_id` — выполняет soft-delete data view по UUID.
- `POST /api/v1/data-views/:data_view_id/restore` — восстанавливает data view по UUID; body отсутствует.
- `DELETE /api/v1/data-views/:data_view_id/hard` — hard-delete при отсутствии dependency usages.

Data view source request example:

```json
{
  "source": { "steps": [] },
  "inputSchema": {},
  "outputSchema": {}
}
```

## 4. Source boundary только для старых моделей

В №20 не придумывать новый source format и не конвертировать существующий JSONB в TEXT без отдельного domain contract.

Для Converters, Queries и Data Views сохранить форматы из задач №2–3 как canonical authoring payload. Изменяется API update semantics:

- PATCH обновляет metadata/relations;
- PUT `/source` атомарно заменяет весь authoring payload;
- list не возвращает тяжёлый authoring payload;
- detail возвращает полный persisted document;
- backend не компилирует и не исполняет source.

Не сохранять AST, IR, diagnostics, compiled graph или runtime state. Source rules новых моделей из задач №4–19 здесь не менять.

## 5. Portable export/import только для моделей №1–3

Добавить `EntityPortableAdapter` из задачи №7 только для:

```text
Project
Folder
Converter
Query
DataView
```

`components_legacy` не получает portable adapter: сначала данные должны быть перенесены в canonical Components. Adapter canonical Components является ответственностью задачи №16 и в №20 не изменяется.

### `POST /api/v1/domain/export`

Экспортирует выбранные модели №1–3 из workspace header без foreign UUID.

Request body:

```json
{
  "entityTypes": ["projects", "folders", "converters", "queries", "data-views"],
  "includeDeleted": false
}
```

Response содержит portable documents и relations по identity.

### `POST /api/v1/domain/import?conflict=fail&atomic=true`

Импортирует portable documents в workspace из header, разрешая relations в UUID целевой database.

Query parameters:

```text
conflict = fail | overwrite | rename
atomic   = true | false
```

Request body содержит export document. Для `rename` request должен содержать explicit rename map. Silent rename/overwrite запрещены.

В этой задаче end-to-end tests должны покрывать только adapters моделей №1–3. Не дорабатывать adapters новых models.

## 6. Общие ошибки

```json
{
  "code": "query_not_found",
  "message": "Query not found",
  "details": {}
}
```

Минимальные codes для scope №1–3:

```text
validation_error
project_not_found
folder_not_found
converter_not_found
query_not_found
data_view_not_found
*_identity_conflict
*_in_use
workspace_required
workspace_not_found
folder_entity_type_mismatch
legacy_component_migration_conflict
legacy_component_reference_unresolved
system_entity_mutation_forbidden
internal_error
```

Не возвращать SQL/internal errors и sensitive values.

## Acceptance Criteria

- scope задачи ограничен моделями, созданными в №1–3;
- новые модели из №4–19 не изменены и не продублированы в implementation;
- все `components_legacy` перенесены без потери source/metadata либо migration безопасно остановлена с report;
- legacy component schema/code/routes полностью удалены только после проверки migration;
- `/api/v1/components` остаётся единственным component API и не переделывается в №20;
- Projects, Folders, Converters, Queries и Data Views используют `X-Endge-Workspace` и verified `workspaceID`;
- их structured relations хранятся как workspace-scoped UUID foreign keys;
- их mutations выполняются только по UUID, GET detail работает по UUID или identity;
- list/detail и PATCH/PUT semantics соответствуют этому документу;
- hard-delete не каскадно удаляет самостоятельные documents и возвращает `409 *_in_use` при dependencies;
- portable adapters и end-to-end import/export tests добавлены только для Project, Folder, Converter, Query и DataView;
- OpenAPI содержит headers, parameters, request/response examples и errors на русском;
- Payload-specific routes/query conventions для моделей №1–3 отсутствуют;
- `make sqlc`, `make docs` и `go test ./...` проходят.
