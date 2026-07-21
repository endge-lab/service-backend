# 01. Projects + Folders Foundation

## Актуальный scope задачи

Сохранить исходную clean architecture концепцию задачи: migrations, domain entities, repository ports, usecases, HTTP handlers, sqlc и tests для Projects и Folders.

При этом старая трактовка Project как владельца всех документов больше не действует. Project — workspace-scoped structural coordinate и второй слой configuration cascade. Одна модель или source document может использоваться несколькими проектами, поэтому общий domain document не получает `project_id` только ради принадлежности проекту.

Актуальная цепочка конфигурации:

```text
Workspace -> Project -> Environment -> Tenant
```

Задача зависит от `04-workspaces-foundation` и `05-workspace-context-middleware`. Если старая реализация этой задачи уже была выполнена, её schema/API нужно привести к contract ниже; отметка `DONE` в имени файла относится к предыдущей модели.

## Главное правило идентификаторов

Все UUID `id` генерируются backend.

Клиент передаёт стабильные identities:

- `projectIdentity` для ссылки на Project;
- `folderIdentity` и `parentIdentity` для Folders;
- `navigationIdentity` для project navigation;
- `allowedEnvironmentIdentities` для allow-list environments.

UUID возвращаются как technical ids, но public API не требует вручную передавать UUID для создания и связывания сущностей.

## Project model

Project принадлежит Workspace, но не Tenant и не Environment. Environment выбирается отдельно в execution context; Project лишь ограничивает допустимый список.

```sql
CREATE TABLE projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  identity TEXT NOT NULL,
  display_name TEXT NOT NULL,
  navigation_id UUID NULL,
  folder_id UUID NULL,
  deleted_at TIMESTAMPTZ NULL,
  configuration JSONB NOT NULL DEFAULT '{"mode":"inherit","patch":{}}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, identity),
  CHECK (btrim(identity) <> ''),
  CHECK (btrim(display_name) <> ''),
  CHECK (jsonb_typeof(configuration) = 'object')
);
```

`navigation_id` после создания Navigations связывается foreign key с `ON DELETE SET NULL`. Navigation обязана принадлежать тому же Workspace.

`folder_id` указывает на folder с `entity_type=projects` в том же Workspace. При отсутствии folder используется `root-projects`.

Не добавлять в Projects поля `tenant_id` или `environment_id`.

### Allowed environments

Many-to-many relation хранить отдельно:

```sql
CREATE TABLE project_allowed_environments (
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE RESTRICT,
  PRIMARY KEY (project_id, environment_id)
);
```

Usecase проверяет, что Project и Environment принадлежат одному Workspace. Пустой allow-list означает, что Project не ограничивает выбор Environment. Непустой список разрешает только указанные environments.

## Project configuration contribution

Project хранит `EndgeConfigurationContribution`, а не полную effective configuration.

Default:

```json
{
  "mode": "inherit",
  "patch": {}
}
```

Пример project-specific theme и SFC adapter:

```json
{
  "mode": "inherit",
  "patch": {
    "themes": {
      "entries": [
        {
          "key": "airport",
          "op": "upsert",
          "value": {
            "identity": "airport",
            "displayName": "Airport"
          }
        }
      ]
    },
    "defaultTheme": {
      "op": "set",
      "value": "airport"
    },
    "sfcAdapterIds": {
      "entries": [
        {
          "key": "shadcn-vue",
          "op": "upsert",
          "value": "shadcn-vue"
        }
      ]
    }
  }
}
```

`inherit.patch` поддерживает:

```text
collection operations:
  vars, locales, themes, sfcAdapterIds
  entries: { key, op: upsert, value } | { key, op: remove }

scalar operations:
  sse, defaultLocale, fallbackLocale, defaultTheme,
  defaultAuthProfileIdentity, defaultSfcAdapterId
  value: { op: set, value } | { op: remove }
```

Required scalar fields нельзя удалить. Для `defaultAuthProfileIdentity` remove означает `null`; для optional `sse` — отсутствие SSE settings.

`mode=replace` содержит полную `EndgeConfiguration` из задачи №4 и сбрасывает Workspace configuration для этого и последующих layers.

Backend валидирует contribution, но не сохраняет result применения к Workspace. Effective configuration вычисляется только для полного execution context.

## Folder model

Folder — workspace-scoped дерево навигации по entity type. Folder больше не принадлежит Project.

```sql
CREATE TABLE folders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  entity_type TEXT NOT NULL,
  identity TEXT NOT NULL,
  display_name TEXT NOT NULL,
  parent_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
  is_root BOOLEAN NOT NULL DEFAULT FALSE,
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, entity_type, identity),
  CHECK (btrim(identity) <> ''),
  CHECK (btrim(display_name) <> '')
);
```

Не добавлять `project_id` в Folders. Isolation обеспечивается через `workspace_id + entity_type`.

Минимально поддержать entity types, необходимые ближайшим задачам:

```text
projects
components
converters
queries
data-views
```

Новые task files могут расширять список. Tenant и Environment tasks добавляют `tenants` и `environments`.

Ограничения:

- в Workspace и `entity_type` существует ровно одна system root folder;
- `parent_id` ссылается только на folder того же Workspace и `entity_type`;
- folder не может быть родителем самой себе или своему ancestor;
- root folder нельзя переместить или удалить;
- system folder нельзя hard-delete.

Для Projects создать root:

```text
identity: root-projects
entityType: projects
isRoot: true
isSystem: true
parentIdentity: null
```

Root provisioning выполняется при создании Workspace или идемпотентным bootstrap usecase. Создание каждого Project не создаёт отдельный набор root folders.

## Project behavior

Create Project выполняет в одной transaction:

1. resolve Workspace из request context;
2. validate unique project identity;
3. resolve `root-projects` или explicit project folder;
4. resolve optional navigation;
5. validate `configuration` или поставить clean inherit default;
6. создать Project;
7. resolve и сохранить allowed environments.

Soft delete заполняет `deleted_at`. List/get по умолчанию не возвращают deleted Projects. Restore очищает `deleted_at`. Hard delete удаляет Project и allow-list, но не удаляет shared environments, navigation, folders или domain documents.

## API

Project and Folder endpoints требуют:

```http
X-Endge-Workspace: default
```

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
  "identity": "groundhandling",
  "displayName": "Ground Handling",
  "navigationIdentity": "groundhandling-main",
  "folderIdentity": "root-projects",
  "allowedEnvironmentIdentities": ["dev", "production"],
  "configuration": {
    "mode": "inherit",
    "patch": {}
  }
}
```

`navigationIdentity`, `folderIdentity`, `allowedEnvironmentIdentities` и `configuration` optional. Default configuration — clean inherit; default folder — `root-projects`; empty allow-list — без environment restrictions.

Response:

```json
{
  "id": "00000000-0000-4000-8000-000000000101",
  "identity": "groundhandling",
  "displayName": "Ground Handling",
  "navigationIdentity": "groundhandling-main",
  "folderIdentity": "root-projects",
  "allowedEnvironmentIdentities": ["dev", "production"],
  "deletedAt": null,
  "configuration": {
    "mode": "inherit",
    "patch": {}
  },
  "createdAt": "2026-07-16T10:00:00Z",
  "updatedAt": "2026-07-16T10:00:00Z"
}
```

PATCH is partial. `identity`, `id`, Workspace и timestamps неизменяемы. Переданный `configuration` заменяет complete contribution object.

### Folders

```text
GET    /api/v1/folders?entity_type=projects
POST   /api/v1/folders
GET    /api/v1/folders/:folder_identity?entity_type=projects
PATCH  /api/v1/folders/:folder_identity?entity_type=projects
DELETE /api/v1/folders/:folder_identity?entity_type=projects
```

Create request:

```json
{
  "entityType": "projects",
  "identity": "airport-projects",
  "displayName": "Airport Projects",
  "parentIdentity": "root-projects"
}
```

Response:

```json
{
  "id": "00000000-0000-4000-8000-000000000011",
  "entityType": "projects",
  "identity": "airport-projects",
  "displayName": "Airport Projects",
  "parentIdentity": "root-projects",
  "isRoot": false,
  "isSystem": false,
  "createdAt": "2026-07-16T10:00:00Z",
  "updatedAt": "2026-07-16T10:00:00Z"
}
```

## Errors

```text
validation_error
configuration_invalid
project_not_found
project_identity_conflict
project_environment_mismatch
environment_not_found
navigation_not_found
navigation_workspace_mismatch
folder_not_found
folder_workspace_mismatch
folder_entity_type_mismatch
folder_cycle
system_folder_delete_forbidden
internal_error
```

## Слои

Реализовать через текущую clean architecture:

- domain entities: `RProject`, `RFolder`;
- ports: `ProjectsRepository`, `FoldersRepository`, `TxManager`;
- postgres repositories и sqlc;
- usecases с identity resolution;
- HTTP handlers и camelCase DTO;
- bootstrap registration through `fx`;
- migrations.

Usecase layer не импортирует postgres или Fiber packages.

## Tests

Минимально нужны:

- validation project/folder input;
- default Project contribution;
- inherit/replace contribution validation;
- Project identity uniqueness inside Workspace;
- allowed environment validation and cross-workspace rejection;
- empty allow-list semantics;
- folder workspace/type checks and cycle prevention;
- Project soft-delete/restore/hard-delete;
- handler tests для API operations;
- `go test ./...`.

## Acceptance Criteria

- Project и Folder являются workspace-scoped;
- Project не владеет shared domain documents и не требует их `project_id`;
- Project не содержит tenant/environment foreign keys;
- Project configuration хранится как contribution с clean inherit default;
- allowed environments реализованы явной many-to-many relation;
- Folders не содержат `project_id`;
- при создании Workspace существует `root-projects`, а создание Project не создаёт отдельные folder trees;
- public relations принимают stable identities, repository хранит UUID;
- soft-deleted Projects не попадают в обычный list;
- нельзя создать folder cycle или удалить system root;
- migration/repository/usecase/HTTP tests проходят.
