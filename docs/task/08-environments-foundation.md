# 08. Environments Foundation

## Цель

Реализовать `REnvironment` как workspace-scoped environment и третий configuration layer. Environment не принадлежит одному Project и может использоваться несколькими проектами.

Backend открытый; endpoints используют `X-Endge-Workspace` и не проверяют пользователя. Задача зависит от `04-workspaces-foundation`, `05-workspace-context-middleware` и `07-domain-relations-and-portable-import`.

## Поля и таблица

```sql
CREATE TABLE environments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  identity TEXT NOT NULL,
  display_name TEXT NOT NULL,
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
  configuration JSONB NOT NULL DEFAULT '{"mode":"inherit","patch":{}}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (workspace_id, identity),
  CHECK (btrim(identity) <> ''),
  CHECK (btrim(display_name) <> ''),
  CHECK (jsonb_typeof(configuration) = 'object')
);
```

API fields:

```text
id
identity
displayName
isSystem
folderIdentity
configuration
createdAt
updatedAt
```

Frontend field `name` — alias для `displayName`, отдельная колонка не нужна.

## Configuration contribution

Environment хранит тот же `EndgeConfigurationContribution`, что Project и Tenant.

Default:

```json
{
  "mode": "inherit",
  "patch": {}
}
```

Пример environment-specific variables:

```json
{
  "mode": "inherit",
  "patch": {
    "vars": {
      "entries": [
        {
          "key": "ENDPOINT_API",
          "op": "upsert",
          "value": {
            "name": "ENDPOINT_API",
            "defaultValue": "https://api.production.example"
          }
        },
        {
          "key": "DEVTOOLS_ENABLED",
          "op": "remove"
        }
      ]
    },
    "sse": {
      "op": "set",
      "value": {
        "url": "https://sse.production.example",
        "authMode": "inherit"
      }
    }
  }
}
```

`mode=replace` содержит полную `EndgeConfiguration` из задачи №4 и полностью сбрасывает Project/Workspace result.

Backend обязан валидировать contribution, но не вычисляет и не сохраняет effective configuration в environment record.

## Execution semantics

Environment применяется после Project:

```text
Workspace -> Project -> Environment -> Tenant
```

Environment — выбранная structural coordinate boot lifecycle. Изменение environment требует нового build context и полной перекомпиляции проекта. Это не live preference.

Project может ограничить допустимые environments через `allowedEnvironments`, но Environment не хранит обратный `project_id` и не дублируется для каждого проекта.

## Folders

Добавить folder type `environments` и идемпотентный system root `root-environments` в каждом workspace.

Папка обязана принадлежать текущему workspace и иметь `entity_type = environments`. При create без `folderIdentity` используется `root-environments`.

## Usecase

```text
Create(workspaceID, input) -> environment
List(workspaceID, folderIdentity?) -> environments
GetByIdentity(workspaceID, identity) -> environment
Update(workspaceID, identity, patch) -> environment
HardDelete(workspaceID, identity) -> void
```

`identity` immutable. System environment нельзя изменять или удалять через public API. `configuration` заменяется как один complete contribution object при PATCH.

## HTTP API

Все endpoints требуют:

```http
X-Endge-Workspace: default
```

```text
GET    /api/v1/environments?folder_identity=root-environments
POST   /api/v1/environments
GET    /api/v1/environments/:environment_identity
PATCH  /api/v1/environments/:environment_identity
DELETE /api/v1/environments/:environment_identity
```

Create request:

```json
{
  "identity": "production",
  "displayName": "Production",
  "folderIdentity": "root-environments",
  "configuration": {
    "mode": "inherit",
    "patch": {}
  }
}
```

Patch request:

```json
{
  "displayName": "Production EU",
  "folderIdentity": "root-environments",
  "configuration": {
    "mode": "inherit",
    "patch": {
      "defaultTheme": {
        "op": "set",
        "value": "dark"
      }
    }
  }
}
```

Response:

```json
{
  "id": "00000000-0000-4000-8000-000000000401",
  "identity": "production",
  "displayName": "Production",
  "isSystem": false,
  "folderIdentity": "root-environments",
  "configuration": {
    "mode": "inherit",
    "patch": {}
  },
  "createdAt": "2026-07-16T10:00:00Z",
  "updatedAt": "2026-07-16T10:00:00Z"
}
```

## Errors

```text
validation_error
configuration_invalid
environment_not_found
environment_identity_conflict
system_environment_mutation_forbidden
folder_not_found
folder_workspace_mismatch
folder_entity_type_mismatch
environment_in_use
internal_error
```

При удалении environment, разрешённого хотя бы одному Project, вернуть `409 environment_in_use`, пока relation не удалена из project allow-list.

## Tests и Acceptance Criteria

- все queries содержат `workspace_id`;
- environment не содержит `project_id`;
- default contribution — `{ mode: "inherit", patch: {} }`;
- repository выполняет JSON round-trip без materialization;
- валидируются inherit/replace и patch operations;
- system environment нельзя изменить или удалить;
- folder scope/type проверяется;
- environment, используемый project allow-list, нельзя удалить неявно;
- API описан в OpenAPI;
- migration/repository/usecase/HTTP tests реализованы;
- `go test ./...` проходит.
