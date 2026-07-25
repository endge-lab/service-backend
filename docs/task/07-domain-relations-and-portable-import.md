# 07. Domain Relations and Portable Import

## Цель

Не только описать правила, а реализовать общую backend foundation для связей между доменными сущностями, dependency usages и будущего переносимого import/export.

После этой задачи существующие и новые usecases должны переиспользовать один механизм разрешения relations и проверки зависимостей. Database обеспечивает referential integrity, а HTTP и LowCode formats остаются переносимыми между environments.

Задача зависит от `04-workspaces-foundation` и `05-workspace-context-middleware`: каждый lookup должен получать проверенный `workspaceID` из context.

Backend открытый, но отсутствие authorization не отменяет validation, foreign keys и безопасное удаление.

## Что разработчик обязан реализовать в этой задаче

### 1. Общий usecase-level relation resolver

Создать package `internal/usecase/relations`, не зависящий от Fiber, HTTP transport и PostgreSQL.

На этом этапе реализовать resolvers только для уже доступных repositories:

```text
ResolveProjectID(ctx, workspaceID, projectIdentity)
ResolveFolderID(ctx, workspaceID, folderIdentity, expectedEntityType, projectID?)
```

Перевести существующие Queries, DataViews, Converters и другие не-legacy usecases с собственной логики project/folder lookup на этот package. Legacy Components специально не рефакторить: они удаляются в задаче №20.

Когда в задачах №10 и №18 появятся соответствующие repositories, в этот же package добавить:

```text
ResolveTypeID(ctx, workspaceID, typeIdentity)
ResolveAuthProfileID(ctx, workspaceID, authProfileIdentity)
```

Правила resolver:

- trim и проверка identity выполняются до repository call;
- lookup всегда содержит `workspace_id`;
- пустая required identity возвращает `validation_error`;
- если identity не найдена в текущем workspace, вернуть `*_not_found`;
- не выполнять дополнительный global lookup только ради `*_workspace_mismatch`;
- `*_workspace_mismatch` использовать только для internal/batch UUID input или обнаруженных повреждённых данных;
- folder дополнительно проверяется по `entity_type` и project;
- raw foreign UUID из public body не принимать;
- HTTP handler не выполняет relation lookup самостоятельно.

Ports должны оставаться domain-oriented. Не создавать один generic repository вида `GetEntity(table, identity)`, который скрывает тип relation и domain errors.

### 2. Dependency index для identities внутри source/JSON

Добавить следующую свободную forward migration — сейчас это `000022_init_domain_dependencies.sql`. Не перенумеровывать и не переписывать уже существующие migrations.

Таблица:

```sql
CREATE TABLE domain_dependencies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  owner_type TEXT NOT NULL,
  owner_id UUID NOT NULL,
  dependency_type TEXT NOT NULL,
  dependency_identity TEXT NOT NULL,
  source_path TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CHECK (btrim(owner_type) <> ''),
  CHECK (btrim(dependency_type) <> ''),
  CHECK (btrim(dependency_identity) <> ''),
  CHECK (btrim(source_path) <> ''),

  UNIQUE (
    workspace_id,
    owner_type,
    owner_id,
    dependency_type,
    dependency_identity,
    source_path
  )
);

CREATE TABLE domain_dependency_states (
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  owner_type TEXT NOT NULL,
  owner_id UUID NOT NULL,
  owner_identity TEXT NOT NULL,
  verification_state TEXT NOT NULL,
  verification_error TEXT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  PRIMARY KEY (workspace_id, owner_type, owner_id),
  CHECK (btrim(owner_type) <> ''),
  CHECK (btrim(owner_identity) <> ''),
  CHECK (verification_state IN ('verified', 'unverified'))
);

ALTER TABLE domain_dependencies
ADD CONSTRAINT domain_dependencies_owner_state_fkey
FOREIGN KEY (workspace_id, owner_type, owner_id)
REFERENCES domain_dependency_states(workspace_id, owner_type, owner_id)
ON DELETE CASCADE;
```

Добавить indexes:

```text
(workspace_id, dependency_type, dependency_identity) — поиск usages
(workspace_id, owner_type, owner_id)                — rebuild/cleanup owner dependencies
```

`owner_id` не может ссылаться обычным FK на domain table, потому что owner относится к разным таблицам. Он ссылается на owner state; целостность связи state с реальным document обеспечивает usecase transaction. При update/delete owner его dependency state заменяется или удаляется в той же transaction, а rows удаляются через owned `CASCADE`.

Отдельная `domain_dependency_states` нужна, чтобы document без найденных references всё равно мог быть помечен как `unverified`. Не создавать fake dependency с типом `unknown`.

Добавить:

```text
internal/domain/entities/domain_dependency.go
internal/usecase/ports/domain_dependencies.go
internal/usecase/dependencies/
internal/repo/postgres/domain_dependencies_repository.go
internal/repo/postgres/queries/domain_dependency.sql
internal/repo/postgres/mappers/domain_dependencies.go
```

Repository/usecase operations:

```text
ReplaceForOwner(ctx, workspaceID, owner, dependencies, verificationState)
DeleteForOwner(ctx, workspaceID, ownerType, ownerID)
ListUsages(ctx, workspaceID, dependencyType, dependencyIdentity)
EnsureNotReferenced(ctx, workspaceID, dependencyType, dependencyIdentity)
```

`ReplaceForOwner` сначала upsert-ит owner verification state, затем удаляет старые dependency rows и вставляет новые. Операция выполняется через существующий `TxManager.WithinTransaction` вместе с сохранением canonical document.

### 3. Контракт извлечения dependencies

Создать typed interface, который entity-specific code сможет реализовать без generic reflection:

```text
DependencyExtractor.Extract(document) -> DependencyExtractionResult
```

`DependencyExtractionResult` содержит `references`, общий `verificationState` и optional `verificationError`. Каждый `DependencyReference` содержит:

```text
dependencyType
dependencyIdentity
sourcePath
```

В этой задаче реализовать infrastructure и tests с test extractor. Реальные extractors подключаются в задачах конкретных моделей:

```text
Types        → schema.fields[].type
Filters      → fields[].vocabIdentity / converterIdentities[]
Components  → references, доступные parser/compiler
Compositions/Stores → references, доступные parser/compiler
```

Если backend не умеет разобрать конкретный source, разрешить сохранение work-in-progress document и сохранить owner state как `unverified`, даже если список найденных dependencies пуст. Нельзя возвращать успешную полную validation, если source фактически не проверялся.

### 4. Проверка перед hard-delete

Перед hard-delete entity usecase вызывает `EnsureNotReferenced`. Если usages существуют:

```text
HTTP status: 409 Conflict
error code: <entity>_in_use
details.usages: первые 20 usages
details.total: полное количество usages
```

FK с `ON DELETE RESTRICT` остаётся последней защитой от race condition. Dependency index нужен для references внутри source/JSON, которые невозможно защитить обычным FK.

### 5. Read-only API для просмотра usages

Добавить endpoint. Редактировать `domain_dependencies` через public API запрещено: это derived projection.

#### `GET /api/v1/domain/usages`

Возвращает документы, которые используют указанную identity внутри source или authoring JSON. Требует `X-Endge-Workspace`.

Query parameters:

```text
dependency_type     = type
dependency_identity = Orders
limit               = 50 (optional, default 50, max 200)
offset              = 0  (optional, default 0)
```

Пример:

```http
GET /api/v1/domain/usages?dependency_type=type&dependency_identity=Orders&limit=50&offset=0
X-Endge-Workspace: default
```

Response `200 OK`:

```json
{
  "items": [
    {
      "ownerType": "type",
      "ownerId": "550e8400-e29b-41d4-a716-446655440000",
      "ownerIdentity": "OrderList",
      "sourcePath": "schema.fields[0].type",
      "verificationState": "verified"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

Если required query parameters отсутствуют или пусты — `400 validation_error`. Пустой result возвращает `200` и `items: []`.

### 6. Foundation для portable export/import

Создать package `internal/usecase/portable` с transport-independent contracts:

```text
PortableDocument
PortableRelation
ImportOptions
ImportResult
EntityPortableAdapter
```

Минимальные правила:

- `PortableDocument` содержит собственную `identity`, но не foreign UUID;
- structured relations представлены как `entityType + identity`;
- canonical source/JSON сохраняется без замены identities/tags;
- import planner строит map `entityType + identity -> target UUID`;
- supported conflict policies: `fail`, `overwrite`, `rename`;
- `rename` работает только с явно переданной новой identity;
- atomic mode выполняется через `TxManager.WithinTransaction` и не оставляет partial graph.

В №7 реализовать contracts, registry/planner и unit tests с fake adapters. Не реализовывать здесь полноценный import всех моделей: их tables/repositories ещё отсутствуют.

Public export/import endpoints в этой задаче не создавать. Entity adapters новых моделей добавляются в их собственных задачах №8–19. В задаче №20 добавляется HTTP transport над готовым registry и adapters только для старых моделей из №1–3; новые модели там повторно не дорабатываются.

## Что реализуется не в №7, а в задачах моделей

Каждая задача №8–19 обязана отдельно:

1. Добавить конкретные `*_id UUID` columns и foreign keys.
2. Добавить `UNIQUE (workspace_id, id)` для target table, если нужен composite FK.
3. Подключить typed relation resolver для своих HTTP `...Identity` fields.
4. Реализовать entity-specific `DependencyExtractor`, если references живут в source/JSON.
5. Обновлять dependency projection в одной transaction с document.
6. Вызывать delete guard перед hard-delete.
7. Добавить свой `EntityPortableAdapter` в registry.
8. Добавить entity-specific relation/import tests.

Задача №7 не должна заранее создавать пустые repositories для ещё не существующих Types, AuthProfiles и других будущих models.

## Классификация ссылок

### Structured database relation

Если поле однозначно ссылается на известную таблицу и используется usecase/repository, в БД хранить UUID foreign key:

```text
project_id
folder_id
query_id
navigation_id
auth_profile_id
input_type_id
output_type_id
```

Public HTTP принимает и возвращает human-readable identity:

```json
{
  "projectIdentity": "demo-project",
  "folderIdentity": "root-components",
  "authProfileIdentity": "main-keycloak"
}
```

Usecase разрешает identity в UUID внутри текущего workspace и только после этого вызывает repository.

### LowCode authoring reference

References внутри canonical source или вложенного authoring JSON остаются строковыми identities/tags:

```text
Component/Composition/Store source
Filter.fields[].vocabIdentity
Filter.fields[].converterIdentities
Type.schema.fields[].type
component identity/tag references
```

Их нельзя заменять UUID внутри source: document должен быть readable and portable. Такие references обслуживаются dependency extractor/index, а не обычным FK.

### External reference

References на внешние registries, secrets и code providers остаются строками:

```text
credentialRefs
codeRef
collectionSlug
SFC adapter ids
```

Backend валидирует format и не пытается создать foreign key на внешний ресурс.

## Database integrity

Для защиты от cross-workspace relation применять composite foreign key:

```sql
ALTER TABLE auth_profiles
ADD CONSTRAINT auth_profiles_workspace_id_id_unique
UNIQUE (workspace_id, id);

ALTER TABLE vocabs
ADD CONSTRAINT vocabs_auth_profile_workspace_fkey
FOREIGN KEY (workspace_id, auth_profile_id)
REFERENCES auth_profiles(workspace_id, id)
ON DELETE RESTRICT;
```

Такой же принцип применить к project, folder, query, type и другим workspace-scoped relations. Если migration order пока не позволяет добавить FK, создать UUID column в ранней migration и добавить constraint после создания target table.

Deletion policy:

```text
ON DELETE RESTRICT → значимая relation, потеря которой ломает document
ON DELETE SET NULL → optional relation, отсутствие которой валидно
ON DELETE CASCADE  → только owned child, не имеющий смысла без owner
```

`CASCADE` нельзя использовать как способ обойти dependency checks.

Usecase validation и DB CHECK должны согласовываться. Например:

```sql
CHECK (
  (auth_mode = 'profile' AND auth_profile_id IS NOT NULL)
  OR
  (auth_mode <> 'profile' AND auth_profile_id IS NULL)
)
```

## Portable copy rules

### Полное копирование database

`pg_dump/restore` переносит UUID и foreign keys без изменения. Дополнительный mapping не нужен.

### Logical export

Portable document не содержит foreign UUID. Relations экспортируются по identity:

```json
{
  "identity": "airports",
  "authProfileIdentity": "main-keycloak",
  "folderIdentity": "root-vocabs"
}
```

### Logical import

Import выполняется phases:

1. Определить target workspace по identity.
2. Создать или найти base entities по `(workspace_id, identity)`.
3. Построить map `entityType + identity -> target UUID`.
4. Разрешить structured relations и сохранить UUID foreign keys.
5. Сохранить canonical source/JSON identities без переписывания.
6. Выполнить dependency validation и вернуть unresolved references.

Silent overwrite/rename запрещены. Result возвращает `created`, `updated`, `skipped`, `errors`; atomic mode не оставляет partial graph.

## Общие ошибки

```text
validation_error
*_not_found
*_workspace_mismatch
*_in_use
folder_entity_type_mismatch
unresolved_dependency
import_identity_conflict
invalid_relation
```

Frontend может выполнять early validation, но backend/usecase/database остаются authoritative.

## Definition of Done для задачи №7

- создан и используется `internal/usecase/relations` для существующих project/folder relations;
- duplicate project/folder resolving удалён из не-legacy usecases;
- migration `000022_init_domain_dependencies.sql` создаёт dependency и state tables, constraints и indexes;
- entity, port, SQL queries, sqlc code, mapper и PostgreSQL repository для dependencies реализованы;
- dependency usecase умеет transactionally replace/delete projection, искать usages и блокировать hard-delete;
- `GET /api/v1/domain/usages` реализован и описан в OpenAPI на русском;
- package `internal/usecase/portable` содержит contracts, adapter registry и import planner;
- public export/import endpoints пока отсутствуют;
- unit tests покрывают resolver normalization, workspace scope, folder type/project validation, dependency replacement и `*_in_use`;
- repository/integration tests покрывают indexes, uniqueness и transaction rollback;
- portable planner tests покрывают UUID remap, `fail/overwrite/rename` и atomic rollback с fake adapters;
- architecture test запрещает relation lookup из HTTP handlers и raw foreign UUID в public DTO;
- `make sqlc` выполнен, generated code актуален;
- `go test ./...` проходит.
