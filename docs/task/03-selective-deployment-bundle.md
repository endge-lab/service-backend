# Selective deployment bundle: перенос выбранного контекста с зависимостями

## Статус и приоритет

- Статус: обязательная архитектурная основа для межсерверной синхронизации.
- Приоритет: высокий.
- Область изменений: backend contract, хранение dependency manifest и построение immutable bundle.
- Внешняя зависимость: manifest семантических зависимостей формируется компилятором Endge вне этого backend repository.

## Бизнес-контекст

Один Workspace содержит динамические фасеты и много документов разных типов:
Composition, Query, Store, Component, Type, Style и другие сущности. Для одного
запуска нужен конкретный набор выбранных документов фасетов и только достижимая
часть runtime-графа.

Нужно уметь подготовить переносимый пакет только для выбранного контекста, например:

- facet selections `{ "application": "aodb", "customer": "ramax", "stage": "production" }`;
- startup Composition или явно выбранные корневые Composition;
- все документы, от которых они транзитивно зависят;
- требования к установленным integrations.

Пакет должен быть полным для запуска выбранного контекста, но не должен
захватывать несвязанные документы. Имена, количество и порядок фасетов берутся
из release snapshot; backend не кодирует специальные фасеты.

Эта задача не выполняет сетевую синхронизацию и не изменяет target workspace. Она создаёт детерминированный immutable artifact, который следующая задача сможет безопасно сравнить и применить.

## Почему нельзя просто фильтровать полный workspace snapshot

Документы Endge связаны не только SQL foreign keys. Значительная часть зависимостей находится в authored source и известна компилятору:

- Composition использует Query, Store, Component, Computation, Vocab и другие Composition;
- Component может использовать дочерние Component и Type;
- Query может ссылаться на Filter, DataView, Type, Vocab и auth profile;
- Style и Computation также имеют semantic dependencies.

Backend написан на Go и хранит source как строку. Он не должен повторно реализовывать TypeScript/Composition compiler, разбирать source регулярными выражениями или поддерживать второй неполный dependency parser.

Правильная граница:

```text
authored source
      |
      v
внешний Endge compiler
      |
      v
versioned dependency manifest
      |
      v
backend validation + storage + closure + bundle
```

Компилятор владеет смыслом source. Backend владеет доступом, привязкой manifest к commit/release, построением closure, portable contract и аудитом.

## Термины для разработчика backend

- **Document** — workspace-scoped сущность с парой `(type, identity)` и revision.
- **Root** — выбранный документ фасета либо корневая Composition.
- **Dependency edge** — типизированная ссылка одного документа на другой.
- **Dependency manifest** — полный версионированный граф документов на конкретном workspace commit.
- **Closure** — roots плюс все транзитивно достижимые обязательные зависимости.
- **Facet selection** — map `facetIdentity -> documentIdentity`, проверенная по определениям фасетов release.
- **Required integration** — identity/version integration, необходимой bundle; credential и target configuration в пакет не входят.
- **Deployment bundle** — immutable JSON с выбранным контекстом, closure, provenance и checksum.

## Текущее состояние backend

Изучить перед реализацией:

- `internal/usecase/workspace_state/coordinator.go` — список поддерживаемых collections;
- `internal/repo/postgres/releases_repository.go` — export workspace на commit head;
- `internal/domain/entities/portable.go` — полный `PortableBundle`;
- `internal/usecase/workspace_state/import.go` — валидация полного snapshot;
- `internal/usecase/workspace_state/support.go` — structured relations и portable normalization;
- `internal/domain/entities/commit.go` — commit и commit changes;
- `internal/domain/entities/release.go` — immutable release;
- `internal/api/http/v1/composition/transport.go` — Composition source и kind;
- `migrations/000030_workspace_commits.sql`;
- `migrations/000031_document_revisions.sql`;
- `migrations/000034_releases.sql`.

Существующий `PortableBundle`:

- всегда представляет полный workspace snapshot;
- не содержит revisions/commits;
- не содержит dependency graph;
- включает workspace profile;
- предназначен для полного ревизионного import с обратимым soft-delete отсутствующих документов.

Существующая `validateSnapshotRelations` проверяет только ограниченные
backend-visible связи: facets/facet-documents, folders, workspace startup
Composition, update-store и vocab-auth-profile. Этого недостаточно для semantic
closure.

## Цель

После реализации backend умеет:

1. принять dependency manifest от внешнего compiler при создании workspace commit;
2. проверить структуру manifest и связать его с точным commit;
3. скопировать immutable manifest в release при создании release;
4. по release и selection построить roots;
5. дополнить semantic graph структурными backend relations;
6. вычислить полное транзитивное closure;
7. показать plan без создания/изменения target state;
8. экспортировать детерминированный `deployment-bundle` без UUID, пользователей и секретов;
9. отказать, если полноту пакета нельзя доказать.

## Не входит в задачу

- сетевое подключение к другому backend;
- применение bundle к target workspace;
- conflict resolution между source и target;
- перенос raw rows `document_revisions` или `workspace_commits`;
- автоматическое изменение source code при identity mapping;
- реализация Endge compiler в Go;
- чтение frontend-кода во время runtime backend;
- перенос credentials, token, password, DSN или integration configuration;
- создание Git branches.

## Основной принцип: fail closed

Selective bundle нельзя считать корректным, если:

- у source commit/release нет dependency manifest;
- manifest имеет неподдерживаемую schema version;
- manifest не соответствует документам commit;
- обязательная dependency отсутствует;
- root отсутствует;
- selection содержит неизвестный фасет или недоступный документ фасета;
- selection пропускает активный фасет, для которого release содержит документы;
- обнаружен secret-bearing payload;
- compiler пометил artifact как некомпилируемый или manifest incomplete.

В таких случаях API возвращает явную ошибку/невалидный plan. Нельзя молча включить только известные документы и назвать пакет полным.

## Этап 1. Dependency manifest contract

### Привязка к commit

Расширить создание commit так, чтобы request мог содержать dependency manifest для состояния на `expectedHeadSequence`.

Backward compatibility:

- обычный commit без manifest по-прежнему можно создать;
- такой commit можно просматривать и восстанавливать;
- из него нельзя создать selective deployment bundle;
- создание release из commit без manifest допустимо только если release не используется для selective deployment; API bundle затем возвращает `409 dependency_manifest_missing`.

Не пытаться сгенерировать пустой manifest на backend.

### Предлагаемый JSON contract v1

```json
{
  "schemaVersion": 1,
  "complete": true,
  "headSequence": 42,
  "nodes": [
    {
      "documentType": "compositions",
      "identity": "schedule-page",
      "revision": 7,
      "compilable": true,
      "dependencies": [
        {
          "targetKind": "document",
          "documentType": "queries",
          "identity": "schedule",
          "role": "composition-query",
          "required": true
        },
        {
          "targetKind": "document",
          "documentType": "components",
          "identity": "schedule-table",
          "role": "child-component",
          "required": true
        }
      ]
    }
  ],
  "diagnostics": []
}
```

Правила v1:

- `schemaVersion` обязателен и равен `1`;
- `complete=true` означает, что compiler обработал весь workspace state, а не только открытый документ;
- точнее, `complete=true` покрывает все коллекции, semantic dependencies которых принадлежат compiler. Facets, facet-documents, folders и другие чисто структурные документы backend добавляет отдельными typed edges;
- `headSequence` равен ожидаемому head commit;
- node key — `(documentType, identity)`;
- `revision` — backend revision документа, увиденная compiler producer;
- `compilable=false` запрещает deployment для closure, содержащего node;
- dependency имеет `targetKind`, typed target, role и required flag;
- `targetKind=document` требует `documentType` и `identity`;
- `targetKind=integration` требует `identity` и точную `version`, но не configuration;
- `targetKind=builtin` требует стабильную builtin identity и не ищется в workspace documents;
- `role` диагностический, не определяет target type;
- unknown поля допустимы только по заранее выбранной forward-compatibility политике. Для v1 предпочтительно strict decoding;
- размер manifest ограничить. Лимит должен быть существенно меньше общего body limit и конфигурироваться/документироваться;
- duplicate nodes и duplicate edges запрещены;
- empty `identity`/unknown collection запрещены;
- `diagnostics` не должны содержать source или секреты. Хранить только code, severity, document ref и короткое сообщение.

Не принимать backend UUID как portable identity manifest. UUID другой инсталляции непереносим.

### Backend validation manifest

В одной транзакции создания commit backend обязан:

1. проверить `manifest.headSequence == expectedHeadSequence`;
2. проверить `complete=true`;
3. проверить schema version;
4. проверить уникальность nodes/edges;
5. для каждого node найти document `(workspace_id, type, identity)` на commit head;
6. сверить revision;
7. проверить, что required `targetKind=document` node существует в manifest; builtin/integration проверить по отдельным typed правилам;
8. проверить отсутствие cross-workspace ссылок;
9. canonicalize manifest и вычислить SHA-256;
10. сохранить manifest только если commit успешно создан.

Если commit transaction откатывается, manifest не должен оставаться.

Не запрещать циклы общим правилом backend: некоторые будущие типы графа могут допускать рекурсивные ссылки. Closure обязан использовать visited set. Флаг `compilable` и compiler diagnostics являются authoritative для semantic validity.

## Хранение manifest

Добавить новую forward migration. Не изменять применённые migration.

Рекомендуемая таблица:

```text
workspace_commit_dependency_manifests
  commit_id              UUID PK/FK -> workspace_commits ON DELETE CASCADE
  schema_version         INTEGER NOT NULL
  checksum               TEXT NOT NULL
  data                   JSONB NOT NULL
  created_by             UUID NOT NULL -> service_users
  created_at             TIMESTAMPTZ NOT NULL
```

Manifest immutable вместе с commit.

В `releases` добавить nullable immutable поля:

```text
dependency_manifest_schema_version INTEGER
dependency_manifest_checksum       TEXT
dependency_manifest                JSONB
```

При создании release из commit:

- полный workspace snapshot формируется как сейчас;
- manifest копируется из source commit в release;
- release остаётся self-contained и не зависит от дальнейшей доступности commit manifest;
- существующие releases получают `NULL` и не поддерживают selective bundle.

Почему manifest нужно копировать в release: release является долговременной публикационной границей, commit history может иметь другую retention/purge политику в будущем.

## Этап 2. Формирование roots

API принимает selection:

```json
{
  "facetSelections": {
    "application": "aodb",
    "customer": "ramax",
    "stage": "production"
  },
  "compositionIdentities": ["aodb-main"]
}
```

Для v1:

- `facetSelections` содержит не более одного document identity для каждого фасета;
- каждый key должен указывать на активный фасет release, а значение — на активный документ именно этого фасета;
- для каждого активного фасета с документами selection обязан содержать значение;
- порядок применения не передаётся клиентом: backend использует `RFacet.position` из release;
- `compositionIdentities` содержит одну или несколько активных Composition;
- если список Composition не передан, root берётся из `workspace.startupCompositionIdentity` release;
- пустой или неизвестный root делает plan невалидным.

Initial roots:

1. все документы из `facetSelections` вместе с определениями их фасетов;
2. явно выбранные Composition либо startup Composition Workspace;
3. Workspace configuration, необходимая для разрешения effective configuration.

Остальные Composition не добавляются автоматически. Они попадут только как
dependencies, достижимые из корневых Composition.

Backend не выводит root из `kind` и не угадывает «главную страницу» по имени.

## Этап 3. Единый dependency graph

Backend строит graph из двух источников, не создавая двух независимых closure:

### Semantic edges из compiler manifest

Использовать их без повторного parsing source.

### Structured edges, которыми владеет backend

Добавить edges из typed stored fields:

- document - folder и folder - parent folder;
- facet-document - facet;
- workspace - startup Composition;
- update - store через `storeIdentity`;
- vocab - auth-profile через `authProfileIdentity`;
- document - integration requirement, если manifest/контракт указывает integration identity/version;
- другие typed relations, уже явно представленные backend schema.

Нельзя извлекать зависимости поиском строк по arbitrary JSON/source.

Structured edge и semantic edge с одинаковой source/target парой дедуплицируются, но diagnostic roles можно сохранить.

## Этап 4. Closure

Алгоритм:

1. Положить roots в deterministic queue.
2. Для каждого node добавить его обязательные outgoing dependencies.
3. Повторять до отсутствия новых nodes.
4. Использовать visited set по `(documentType, identity)`.
5. Для каждого включённого документа добавить необходимые ancestor folders.
6. Проверить существование и compilable state каждого included node.
7. Собрать required integrations без configuration/secrets.
8. Отсортировать documents в стабильном порядке: collection order + identity.

Optional dependency:

- не включается автоматически, если нет отдельного основания;
- отображается в plan как optional external/omitted dependency;
- required missing dependency делает plan invalid.

Closure не включает reverse dependants. Если выбранный Query используется другой
недостижимой Composition, эта Composition не должна попасть в source bundle.
Target impact analysis выполняется в задаче синхронизации.

## Этап 5. Deployment bundle contract

Не расширять `PortableBundle` случайными optional полями. Создать отдельный тип, потому что полный workspace import и selective deployment имеют разную семантику.

Предлагаемый envelope:

```json
{
  "kind": "deployment-bundle",
  "schemaVersion": 1,
  "source": {
    "workspaceIdentity": "development",
    "releaseIdentity": "release-2026-08-05",
    "releaseId": "550e8400-e29b-41d4-a716-446655440000",
    "releaseChecksum": "...",
    "sourceCommitId": "550e8400-e29b-41d4-a716-446655440001",
    "headSequence": 42,
    "dependencyManifestChecksum": "..."
  },
  "selection": {
    "facetSelections": {
      "application": "aodb",
      "customer": "ramax",
      "stage": "production"
    },
    "compositionIdentities": ["aodb-main"]
  },
  "documents": {
    "facets": [],
    "facet-documents": [],
    "compositions": [],
    "queries": []
  },
  "requiredIntegrations": [
    { "identity": "http", "version": "1.4.0" }
  ],
  "dependencyGraph": {
    "nodes": [],
    "edges": []
  },
  "checksum": "..."
}
```

`releaseId`/`sourceCommitId` являются provenance, но target не использует их как свои UUID.

Правила содержимого:

- documents используют portable identity и authored data;
- не включать local document UUID;
- не включать user/actor IDs;
- не включать revision UUID и workspace sequence как target history;
- можно включить source revision number/checksum в provenance документа;
- не включать workspace members/RBAC;
- не включать workspace configuration целиком;
- не включать integration configuration;
- не включать secret values;
- credential references допустимы только как имена внешних ссылок, если проходят существующую validation;
- folders включаются только необходимые roots/ancestors, system roots target создаёт/разрешает по действующим правилам;
- пустые collections либо опускаются по чёткой canonical policy, либо всегда сериализуются пустыми. Выбрать один вариант и зафиксировать тестом.

Checksum считается по canonical JSON без поля `checksum`. Нужна одна canonical serialization implementation, используемая plan/export/sync.

## API

### Plan

```text
POST /api/v1/releases/{releaseIdentity}/deployment-bundle/plan
X-Endge-Workspace: <source workspace>
```

Body содержит selection.

Требования доступа: workspace `admin` или `platform_admin`, потому что endpoint раскрывает переносимый код/конфигурацию для другой среды.

Пример response:

```json
{
  "valid": true,
  "releaseIdentity": "release-2026-08-05",
  "selection": {
    "facetSelections": {
      "application": "aodb",
      "customer": "ramax",
      "stage": "production"
    },
    "compositionIdentities": ["aodb-main"]
  },
  "roots": 4,
  "documents": 38,
  "requiredIntegrations": 2,
  "missingDependencies": [],
  "nonCompilableDocuments": [],
  "warnings": [],
  "bundleChecksum": "..."
}
```

Plan не сохраняет target state и не создаёт commit. Bundle строится из immutable release, поэтому source race отсутствует.

### Export

```text
POST /api/v1/releases/{releaseIdentity}/deployment-bundle/export
X-Endge-Workspace: <source workspace>
```

Body содержит ту же selection. POST выбран потому, что selection является структурированным запросом и может расширяться; endpoint остаётся read-only.

Response:

- `Content-Type: application/json`;
- `ETag` равен bundle checksum;
- optional `download=true` создаёт безопасное имя файла;
- одинаковый release + manifest + selection всегда дают одинаковые bytes/checksum;
- invalid plan не экспортируется.

Не создавать случайный timestamp внутри bundle: он разрушит детерминизм. Даты provenance берутся из immutable release, если нужны.

## Ошибки API

Минимальные domain codes:

- `dependency_manifest_missing` — release не имеет manifest;
- `dependency_manifest_version_unsupported`;
- `dependency_manifest_incomplete`;
- `dependency_manifest_mismatch`;
- `deployment_root_not_found`;
- `deployment_facet_selection_invalid`;
- `deployment_dependency_missing`;
- `deployment_document_not_compilable`;
- `deployment_secret_detected`;
- `deployment_bundle_too_large`.

Ошибки должны содержать typed details: document type/identity и dependency target. Не включать source body или secret value.

## Наблюдаемость

Метрики:

- `endge.deployment_bundle.plans_total{result=valid|invalid|error}`;
- `endge.deployment_bundle.documents` histogram;
- `endge.deployment_bundle.bytes` histogram;
- `endge.deployment_bundle.build_duration_ms`;
- `endge.dependency_manifest.validation_total{result=valid|invalid|error}`.

Не использовать workspace/release/document identity в labels.

Логировать только IDs/checksums/counts. Не логировать source и полный bundle.

## Изменения в слоях

Рекомендуемые владельцы:

- `internal/domain/entities/dependency_manifest.go` — контракт manifest;
- `internal/domain/entities/deployment_bundle.go` — bundle/plan;
- `internal/usecase/ports/dependency_manifests.go` — storage port;
- `internal/usecase/deployment_bundles` — roots, graph, closure, canonical bundle;
- `internal/repo/postgres` — manifest storage и historical document lookup;
- `internal/api/http/v1/deployment_bundle` или release subresource handler;
- `internal/bootstrap` — Fx wiring;
- новая Goose migration;
- OpenAPI/README.

Не размещать closure в HTTP handler и не реализовывать его SQL trigger.

## Последовательность реализации

1. Зафиксировать DTO manifest v1 и canonical JSON правила.
2. Добавить migration commit/release manifests.
3. Добавить repository ports и PostgreSQL implementation.
4. Расширить commit create contract и transaction validation.
5. Копировать manifest в release.
6. Реализовать historical lookup документов на release/commit head.
7. Реализовать structured edge providers.
8. Реализовать roots и closure.
9. Реализовать plan.
10. Реализовать deterministic export и checksum.
11. Добавить HTTP API, OpenAPI, docs и metrics.

## Тестовые сценарии

### Manifest

1. Valid complete manifest сохраняется вместе с commit.
2. Commit rollback не оставляет manifest.
3. Неверный head sequence отклоняется.
4. Node с неверной revision отклоняется.
5. Duplicate node/edge отклоняется.
6. Required target отсутствует — manifest invalid.
7. Unsupported schema version отклоняется.
8. Старый commit без manifest остаётся читаемым.
9. Release копирует manifest и checksums.

### Roots и closure

1. Каждый выбранный документ фасета и определение фасета входят в roots.
2. Порядок фасетов берётся из release, а не из request.
3. При отсутствии explicit Composition используется startup Composition Workspace.
4. Невыбранная Composition попадает только через dependency.
5. Транзитивная цепочка Composition-Query-DataView включается полностью.
6. Cycle не вызывает бесконечный обход.
7. Missing required dependency делает plan invalid.
8. Non-compilable included node делает plan invalid.
9. Ancestor folders добавляются, несвязанные folders не добавляются.
10. Несвязанная вторая Composition не попадает в bundle.
11. Integration configuration/credentials отсутствуют в bundle.
12. Unknown facet, документ другого фасета и пропущенный обязательный selection отклоняются.

### Детерминизм и безопасность

1. Разный порядок nodes входного manifest даёт один canonical checksum.
2. Два build одного release/selection byte-identical.
3. UUID пользователей и local document IDs не попадают в JSON.
4. Secret detector отклоняет bundle.
5. Пользователь без admin role не может экспортировать bundle.
6. Одинаковые identities в разных workspace изолированы.
7. Большой bundle корректно ограничивается.

## Критерии приёмки

- Backend не парсит authored source для поиска semantic dependencies.
- Manifest привязан к точному commit head и проверенным revisions.
- Release self-contained хранит immutable copy manifest.
- Bundle строится только из release, а не из меняющегося live workspace.
- Roots и closure имеют один authoritative implementation.
- Несвязанные документы не переносятся.
- Required dependency не может потеряться молча.
- Bundle не содержит secrets, actors, RBAC и target-local integration configuration.
- Export детерминирован и имеет стабильный checksum.
- Существующий полный `PortableBundle` и безопасный ревизионный import продолжают работать независимо от selective bundle.

## Риски и внешние зависимости

- Без producer полного dependency manifest задача не даёт end-to-end deployment. Backend часть должна быть реализована и протестирована независимо, но release будет получать `dependency_manifest_missing`, пока внешний compiler не начнёт отправлять manifest.
- Manifest от клиента нельзя считать истинным только из-за валидного JSON. Backend обязан привязать nodes к commit revisions и отклонять structural mismatch.
- Backend не может доказать semantic completeness лучше, чем external compiler. Поэтому `complete`, version и diagnostics являются частью обязательного контракта.
- Identity mapping с переписыванием arbitrary source не поддерживается. Такой подход повредил бы source и создал скрытый второй compiler.
- Изменение manifest schema требует новой version и явной backward compatibility, а не неявного изменения v1.
