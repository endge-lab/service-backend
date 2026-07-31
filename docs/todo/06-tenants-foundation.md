# 06. Tenants Foundation — история изменений

## Статус

Выполнено. Исходная постановка задачи находится в
`../task/06-tenants-foundation (DONE).md`.

## Реализовано

### Модель, миграция и папки

- Добавлена workspace-scoped сущность `RTenant` без `project_id` и
  `environment_id`: tenant является последним configuration layer, а не
  владельцем Project.
- Таблица `tenants` хранит identity, display name, code, optional description,
  `folder_id`, JSONB contribution и timestamps. Identity и code уникальны в
  границах workspace.
- `folders.entity_type` расширен значением `tenants`; `folder_id` tenant имеет
  FK на folders с `ON DELETE SET NULL`.
- При provisioning workspace создаётся системная корневая папка
  `root-tenants`. Create и Update также идемпотентно обеспечивают её наличие и
  используют её, если `folderIdentity` отсутствует или явно передан `null` в
  PATCH.
- Публичный API принимает только `folderIdentity`; технический `folderId` из
  request body не читается. Папка разрешается в текущем workspace и должна
  иметь entity type `tenants`.

### Configuration contribution

- Добавлены `EndgeConfigurationContribution`, режимы `inherit` и `replace`,
  patch operations `upsert`, `remove` и `set`.
- Create без `configuration` сохраняет чистое наследование:

  ```json
  {"mode":"inherit","patch":{}}
  ```

- Валидируются разрешённые collection/scalar patch keys, операции и обязательные
  scalar fields; `remove` для required scalar отклоняется.
- `replace` проверяется полной validation `EndgeConfiguration`; effective
  configuration не вычисляется и не сохраняется.
- Переданный в PATCH contribution заменяет предыдущий целиком, без скрытого
  merge. Ответы HTTP не раскрывают `sse.manualToken`.

### Repository и usecase

- Добавлены `TenantsRepository`, SQLC CRUD queries и JSONB mappers:
  `Create`, `List`, `GetByIdentity`, `Update`, `HardDelete`.
- Все операции требуют `WorkspaceScope` из context и фильтруют SQL по
  `workspace_id`; tenant другого workspace выглядит как отсутствующий.
- Usecase поддерживает partial PATCH, сохраняет immutable поля (`id`,
  `workspaceID`, `identity`, `createdAt`) и преобразует Folder UUID в public
  `folderIdentity` для response.
- В Repository разграничены конфликты `tenant_identity_conflict` и
  `tenant_code_conflict`; отсутствие tenant возвращает `tenant_not_found`.
- При FK-ограничении во время hard delete PostgreSQL error преобразуется в
  `tenant_in_use` (409). В текущей схеме ещё нет persisted records, ссылающихся
  на tenant, поэтому этот путь подготовлен для последующих таблиц.

### HTTP API и OpenAPI

Добавлен scoped API, использующий `X-Endge-Workspace`:

```text
GET    /api/v1/tenants?folder_identity=root-tenants
POST   /api/v1/tenants
GET    /api/v1/tenants/:tenant_identity
PATCH  /api/v1/tenants/:tenant_identity
DELETE /api/v1/tenants/:tenant_identity
```

- Созданы transport DTO, handler, routes и Fx wiring.
- OpenAPI содержит endpoints, schemas, examples и обязательный workspace
  header; `configuration` в create request помечена как optional.
- Доменные ошибки покрывают `validation_error`, `configuration_invalid`,
  `tenant_not_found`, tenant conflicts, folder errors, `tenant_in_use` и
  `internal_error`.

## Проверки

Добавлены и проходят:

- entity и mapper tests для contribution, nullable fields и JSONB;
- usecase tests для root folder, default contribution, PATCH immutable fields,
  configuration validation и folder error contract;
- repository tests для tenant conflicts, tenant not found и FK delete mapping;
- HTTP tests для всех CRUD routes, optional create configuration, workspace
  header и redaction `sse.manualToken`;
- tagged PostgreSQL integration test для workspace isolation, root folder и CRUD
  tenant repository (требует `TEST_POSTGRES_DSN`).

Выполненная проверка:

```bash
GOCACHE=/tmp/service-backend-go-build go test ./...
git diff --check
```

## Итоговый flow

```text
X-Endge-Workspace
  -> WorkspaceContextMiddleware
  -> context.Context (WorkspaceScope)
  -> tenant HTTP handler
  -> Tenant usecase
  -> TenantsRepository / SQL workspace_id filter
  -> tenant configuration contribution
```

Tenant применяется при boot последним в каскаде
`Workspace -> Project -> Environment -> Tenant`; repository хранит только его
локальный contribution, а не заранее вычисленную effective configuration.
