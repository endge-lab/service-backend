# Межсерверная синхронизация выбранного контекста между средами

## Статус и приоритет

- Статус: стратегическая задача, реализуется после `03-selective-deployment-bundle.md`.
- Приоритет: высокий.
- Область изменений: backend source API, backend target orchestration, service-to-service auth, plan/apply, provenance и audit.
- Первая версия: безопасный pull с three-way conflict detection без автоматического semantic merge source code.

## Бизнес-контекст

Endge может быть развёрнут в нескольких изолированных средах, например:

- development;
- test/staging;
- production;
- отдельные инсталляции разных заказчиков.

Пользователь должен иметь возможность перенести выбранный project/tenant/environment с одного backend на другой. Target backend сам подключается к source backend, скачивает immutable deployment bundle, показывает план и только после явного подтверждения атомарно применяет изменения.

Типичный сценарий:

```text
development backend
  workspace=dev
  release=2026.08.05
  project=aodb
  tenant=ramax
  environment=production
            |
            | authenticated pull
            v
production backend
  workspace=prod
  plan -> review -> apply -> local revisions -> local commit
```

Source не получает право записывать target. Инициатором и владельцем транзакции всегда является target backend.

## Главный архитектурный принцип

Не копировать raw revision/commit rows из source database.

Source `document_revisions` и `workspace_commits` содержат локальные:

- workspace UUID;
- document UUID;
- revision UUID;
- parent revision UUID;
- workspace sequence;
- service user foreign keys;
- commit parent IDs.

Эти значения не имеют той же семантики в target database. Прямой перенос создаст сломанные foreign keys, чужую последовательность истории и несуществующих авторов.

Правильная модель:

1. Source отдаёт immutable deployment bundle и provenance.
2. Target сравнивает bundle со своим live state и последней общей точкой.
3. Target создаёт собственные локальные revisions.
4. Target создаёт один локальный commit с `operation=remote_sync`.
5. В отдельном sync audit сохраняются source instance/workspace/release/checksum и связь с target commit.

Таким образом source history остаётся source history, target history остаётся target history, а provenance связывает их без подмены идентификаторов.

## Зависимость от selective bundle

До начала этой задачи должны существовать:

- `deployment-bundle` schema v1;
- dependency manifest на source release;
- deterministic bundle checksum;
- source plan/export по immutable release и selection;
- secret-free portable documents;
- required integration list.

Remote sync не должен повторно строить dependency closure другим алгоритмом.

## Термины

- **Source** — backend, с которого читается release/bundle.
- **Target** — backend, на котором пользователь создаёт plan и применяет изменения.
- **Remote** — заранее настроенное доверенное подключение target к source.
- **Selection** — source project/tenant/environment roots.
- **Context mapping** — разрешённое отображение source root identities в target root identities.
- **Base** — bundle последней успешно применённой синхронизации для того же remote/mapping.
- **Incoming** — новый source bundle.
- **Local** — текущее состояние target.
- **Conflict** — и source, и target изменили один logical document относительно base несовместимым образом.
- **Sync plan** — короткоживущий immutable план без изменения target.
- **Sync run** — зафиксированная попытка применения и её audit/provenance.

## Цель первой версии

Backend должен поддерживать:

1. заранее настроенные trusted remotes;
2. получение OIDC service token через client credentials;
3. pull deployment bundle с source;
4. строгую проверку schema/checksum/размера;
5. source-target context mapping;
6. dry-run plan с creates/updates/deletes/unchanged/conflicts;
7. three-way conflict detection по последнему успешно применённому base;
8. применение только ранее сохранённого plan с `If-Match` target workspace;
9. локальные revisions и commit `remote_sync`;
10. provenance/audit и идемпотентность;
11. безопасное повторение синхронизации следующего release.

## Не входит в первую версию

- push от source к target;
- двусторонняя фоновая репликация;
- автоматический merge двух разных source code;
- UI для построчного разрешения конфликтов;
- копирование source actors/users/RBAC;
- копирование source credentials и integration configuration;
- создание Git branches;
- синхронизация live workspace без release;
- автоматический rollout без plan/confirmation;
- произвольный URL source в пользовательском request;
- расписание/cron синхронизации.

## Направление связи: только target-initiated pull

Target вызывает source read API. Это обязательное решение первой версии.

Преимущества:

- source credential имеет только read access;
- target сам контролирует момент транзакции;
- target проверяет локальный ETag непосредственно перед apply;
- source не нужно выдавать право записи в production;
- retry и audit находятся у владельца изменяемых данных.

Не добавлять source webhook, который автоматически применяет изменения на target.

## Trusted remote configuration

Пользователь не может передать произвольный URL в `sync/plan`. Иначе backend станет SSRF proxy.

Remote состоит из:

- стабильного `identity`, например `development-eu`;
- отображаемого имени;
- точного HTTPS base URL;
- ожидаемого source instance ID;
- OIDC token URL;
- client ID;
- secret reference;
- audience/scopes;
- connect/request timeout;
- maximum response bytes;
- enabled flag.

### Хранение секретов

Client secret нельзя хранить открытым текстом в таблице и нельзя возвращать через API.

Для первой версии использовать reference-based resolver:

- remote metadata хранит `credential_ref`;
- значение secret поступает из environment/secret mount по этому ref;
- логи содержат только remote identity и credential ref;
- API никогда не возвращает secret;
- отсутствие secret приводит к явной startup/config error или remote unavailable.

Не использовать один общий Configurator browser client secret для межсерверной связи.

### Production validation

- base URL и token URL используют `https`;
- host непустой и canonical;
- userinfo в URL запрещён;
- fragment запрещён;
- redirects выключены либо разрешены только на тот же exact origin. Предпочтительно полностью запретить redirects;
- request URL строится через безопасное присоединение известных path, не string concatenation пользовательского ввода;
- private network адреса допустимы, потому что source может находиться во внутренней сети, но только как заранее настроенный remote;
- DNS rebinding/redirect на другой host не должен обходить allowlist;
- TLS verification нельзя отключить production-флагом «для удобства».

## Stable source instance identity

Source backend должен иметь стабильный `INSTANCE_ID`, уникальный для инсталляции и не меняющийся при restart/redeploy.

Source response содержит instance identity. Target сравнивает её с remote configuration. Несовпадение означает ошибку `sync_remote_instance_mismatch`.

Не использовать hostname/pod name как stable instance ID.

## Service-to-service authentication

Использовать OIDC client credentials:

1. Target получает access token на remote token endpoint.
2. Token кешируется только до безопасного момента перед expiry.
3. Target отправляет `Authorization: Bearer ...` source backend.
4. Source использует существующую bearer validation цепочку issuer/audience/algorithm.
5. Service subject отображается на `service_users` и имеет явный read role в source workspace.

Требования:

- token не логируется;
- token request имеет timeout;
- token cache потокобезопасен;
- при `401` допустим один принудительный refresh и один retry;
- бесконечные auth retries запрещены;
- source endpoint требует workspace access;
- local target user должен иметь `admin`/`platform_admin` target workspace;
- source service account не получает target credential.

Если текущий identity provider не поддерживает client credentials, это внешний blocker. Не заменять его query token или бессрочным shared secret без отдельного решения.

## Source API

Source использует bundle builder из предыдущей задачи.

Рекомендуемый endpoint:

```text
POST /api/v1/releases/{releaseIdentity}/deployment-bundle/export
X-Endge-Workspace: <source workspace>
Authorization: Bearer <service token>
```

Body: source selection.

Дополнительные response headers:

- `ETag: "<bundle checksum>"`;
- `X-Endge-Instance: <stable instance id>`;
- `X-Endge-Bundle-Schema: 1`;
- `Content-Length`, если доступен.

Source endpoint остаётся read-only. Он не создаёт sync run и ничего не знает о target workspace.

Source RBAC: минимум reader доступа к workspace плюс отдельное permission/role policy для export переносимого source. Если текущая модель ролей не имеет отдельного permission, до появления permission использовать `admin/platform_admin` или явно выделенный service principal policy; не открывать export всем authenticated users автоматически.

## Target API

Все target endpoints находятся под обычным auth/current-user/workspace middleware и требуют admin role.

### Создание plan

```text
POST /api/v1/sync/plan
X-Endge-Workspace: <target workspace>
```

Пример request:

```json
{
  "remoteIdentity": "development-eu",
  "sourceWorkspaceIdentity": "development",
  "sourceReleaseIdentity": "release-2026-08-05",
  "selection": {
    "projectIdentity": "aodb",
    "tenantIdentity": "ramax",
    "environmentIdentity": "development"
  },
  "mapping": {
    "projectIdentity": "aodb",
    "tenantIdentity": "ramax",
    "environmentIdentity": "production"
  }
}
```

Backend:

1. авторизует local user и target workspace;
2. находит remote по `remoteIdentity`;
3. получает service token;
4. скачивает source bundle с ограничением времени/размера;
5. проверяет source instance, content type, schema и checksum;
6. проверяет selection/mapping;
7. загружает предыдущий successful base для того же mapping;
8. сравнивает base/incoming/local;
9. выполняет target dependency/integration/impact validation;
10. сохраняет plan и exact incoming bundle на ограниченное время;
11. возвращает summary без изменения target.

### Получение plan

```text
GET /api/v1/sync/plans/{planId}
X-Endge-Workspace: <target workspace>
```

Plan доступен только создавшему пользователю или target admin по выбранной политике. Политика должна быть явной и протестированной.

### Применение plan

```text
POST /api/v1/sync
X-Endge-Workspace: <target workspace>
If-Match: "<target generation>:<target headSequence>"
```

```json
{
  "planId": "550e8400-e29b-41d4-a716-446655440000",
  "confirmation": "production"
}
```

`confirmation` равен identity target workspace.

Backend применяет только bundle, сохранённый внутри plan. Нельзя незаметно повторно скачать «последнюю» версию source между plan и apply.

## Sync plan response

Plan должен быть понятен без знания внутренней схемы:

```json
{
  "planId": "...",
  "valid": false,
  "expiresAt": "2026-08-05T12:30:00Z",
  "targetETag": "\"generation:42\"",
  "source": {
    "instanceId": "endge-dev-eu",
    "workspaceIdentity": "development",
    "releaseIdentity": "release-2026-08-05",
    "bundleChecksum": "..."
  },
  "selection": {},
  "mapping": {},
  "summary": {
    "creates": 8,
    "updates": 12,
    "softDeletes": 1,
    "unchanged": 17,
    "conflicts": 2
  },
  "conflicts": [
    {
      "documentType": "queries",
      "identity": "schedule",
      "reason": "both_changed"
    }
  ],
  "missingIntegrations": [],
  "impactedContexts": [],
  "warnings": []
}
```

Plan с conflicts может быть сохранён и просмотрен, но `valid=false` и apply запрещён.

## Хранение plan

Использовать pattern существующих `workspace_snapshot_import_plans`:

- UUID plan;
- target workspace;
- remote/mapping identity;
- source provenance;
- incoming bundle checksum;
- exact incoming bundle JSONB;
- base reference/snapshot;
- target expected generation/head sequence;
- summary JSONB;
- created_by;
- created_at/expires_at/applied_at.

Default lifetime: 30 минут, конфигурируемо.

Plan:

- одноразовый;
- immutable после создания;
- не применяется после expiry;
- не применяется повторно;
- не применяется другим target workspace;
- не применяется, если target ETag изменился;
- хранит secret-free bundle, но всё равно считается чувствительным source artifact и доступен только admin.

## Provenance и постоянное состояние sync

Добавить forward migrations. Не редактировать существующие applied migrations.

### `workspace_sync_mappings`

Одна логическая линия promotion:

```text
id
target_workspace_id
remote_identity
source_instance_id
source_workspace_identity
source_selection JSONB
target_mapping JSONB
mapping_checksum
last_successful_run_id
created_by / created_at / updated_at
UNIQUE(target_workspace_id, remote_identity, mapping_checksum)
```

### `workspace_sync_runs`

Audit каждой попытки apply:

```text
id
mapping_id
status: applying|succeeded|failed
source_release_identity
source_release_checksum
source_commit_id
source_bundle_checksum
base_bundle_checksum nullable
target_base_head_sequence
target_commit_id nullable
summary JSONB
error_code nullable
initiated_by
created_at / completed_at
```

Не сохранять raw access token/error response body.

### Base state

Для безопасного three-way comparison target должен иметь точное предыдущее applied bundle. Одних checksums недостаточно для определения `source changed`/`local changed` по каждому документу.

Допустимые варианты:

- хранить immutable base bundle JSONB в отдельной таблице по checksum;
- хранить canonical document content/checksum map и при необходимости source content;
- дедуплицировать одинаковые bundle по checksum.

Выбрать self-contained target storage. Не полагаться только на то, что старый source release всегда доступен по сети.

Retention base bundle: пока mapping активен и на него ссылается последняя успешная sync. Старые audit runs могут хранить только checksums/provenance после отдельной retention policy.

## Logical document identity

Сравнение выполняется по typed portable key:

```text
documentType + identity после допустимого context mapping
```

Не сравнивать source и target document UUID.

Canonical content checksum должен исключать target-local поля:

- UUID;
- revision number;
- created/updated actor;
- timestamps;
- workspace sequence;
- committed revision IDs.

Checksum должен включать authored/portable content, active/deleted semantics, folder identity и managed ownership поля, которые реально влияют на состояние.

Использовать одну canonicalization implementation из deployment bundle/domain layer.

## Context mapping v1

Workspace identities source и target могут различаться.

Разрешить явное mapping только roots:

- project identity;
- tenant identity;
- environment identity.

Backend может переписать только typed structural fields:

- identity соответствующего root document;
- Composition `kindIdentity` для mapped owner;
- project `allowedEnvironments` для выбранного mapped environment;
- typed dependency/ownership refs, если их target однозначно является mapped root.

Запрещено:

- search/replace identity внутри arbitrary source string;
- изменение identity обычных Query/Store/Component dependencies в v1;
- неявное mapping по display name;
- автоматическое создание нескольких target environments из одного source environment.

Если корректность требует переписать authored source, plan invalid с `sync_mapping_requires_source_rewrite`.

## Three-way comparison

Для каждого logical document сравнить canonical content:

```text
base     — последняя успешно применённая source версия
incoming — новый source bundle
local    — текущее target состояние
```

### Когда base существует

| Base | Incoming | Local | Результат |
|---|---|---|---|
| A | A | A | unchanged |
| A | B | A | update to B |
| A | A | L | keep local, source unchanged |
| A | B | B | unchanged/already applied content |
| A | B | L, где L != A и L != B | conflict `both_changed` |
| A | отсутствует | A | soft delete |
| A | отсутствует | L != A | conflict `delete_vs_local_change` |
| отсутствует | B | отсутствует | create |
| отсутствует | B | B | unchanged |
| отсутствует | B | L != B | conflict `create_collision` |

### Первый sync без base

- target document отсутствует - create;
- target document byte/content-equivalent - unchanged;
- target document существует с другим content - conflict;
- ничего не удалять, потому что нет доказательства, что target document ранее управлялся этим mapping.

### Важное ограничение merge

Первая версия не объединяет два разных authored source. `both_changed` — конфликт, а не повод выбрать source по принципу last-write-wins.

Это уже безопасная pull/merge модель: backend знает common base и определяет независимые изменения, но автоматический semantic merge откладывается до появления typed merge rules.

## Граница удалений

Sync может soft-delete только документ, который:

1. присутствовал в предыдущем base этого mapping;
2. отсутствует в incoming closure;
3. не изменён локально относительно base;
4. не является system-managed root folder;
5. не управляется другим active mapping.

Нельзя удалять все target documents, отсутствующие в incoming bundle. Bundle представляет выбранный context, а не полный workspace.

Physical purge не входит в sync.

## Shared dependencies и impact analysis

Один Query/Type/Component может использоваться несколькими target contexts. Обновление shared document может повлиять на проекты вне текущего mapping.

Plan обязан построить reverse impact по доступным target dependency manifests/index:

- какие target project/tenant/environment используют изменяемый документ;
- какие из них не входят в текущий mapping;
- какие dependencies останутся валидными после update/delete.

Правила v1:

- update shared document допускается только если нет конфликта, но plan содержит `impactedContexts` warning;
- soft delete shared document, используемого вне mapping, запрещён как conflict `dependency_used_outside_scope`;
- missing target dependency делает plan invalid;
- если target не имеет dependency manifest для impact analysis, fail closed для удаления и показать warning/error для update по выбранной политике. Предпочтительно запретить потенциально опасный update до доказуемого анализа.

Нельзя считать source closure достаточным доказательством безопасности target.

## Integrations и secrets

Deployment bundle содержит только required integration identity/version.

Plan проверяет target:

- integration зарегистрирована;
- установленная/доступная version удовлетворяет точному контракту v1;
- target workspace имеет собственную binding/configuration;
- credential refs разрешаются локально.

Source integration configuration не переносится и не заменяет target configuration.

Если integration отсутствует или version несовместима:

- plan invalid;
- вернуть `missingIntegrations`/`incompatibleIntegrations`;
- не устанавливать integration автоматически в рамках sync.

Прогнать существующую secret validation по incoming bundle после mapping и до сохранения plan.

## Apply transaction

Вся запись target выполняется одной database transaction.

Алгоритм:

1. Загрузить plan по target workspace/user policy.
2. Проверить expiry/applied state.
3. Проверить confirmation.
4. Проверить `If-Match` против сохранённых generation/head sequence.
5. Заблокировать target workspace тем же согласованным механизмом, что snapshot/restore.
6. Повторно проверить live generation/head sequence после lock.
7. Создать `workspace_sync_run(status=applying)` внутри/до транзакции по выбранной audit policy.
8. Создать mutation batch `remote_sync`.
9. Применить creates/updates/soft deletes через общий document lifecycle/repository helpers с typed relation validation.
10. Для каждого изменения записать локальную revision с local actor.
11. Проверить, что pending revisions соответствуют plan.
12. Создать один local commit `operation=remote_sync`, `revisionPolicy=preserve`.
13. Привязать revisions к commit.
14. Обновить mapping/base и run provenance.
15. Пометить plan applied.
16. Commit transaction.

Нельзя вызывать HTTP handlers из sync use case. Нельзя делать отдельную транзакцию на каждый документ.

Добавить `remote_sync` в CHECK constraint `workspace_commits.operation` новой Goose migration. При необходимости добавить допустимую operation для revision/audit layer, не изменяя старые migrations.

Если transaction падает, domain state, commit, mapping/base и applied marker не должны частично сохраниться. Для failed audit run можно использовать отдельную безопасную запись после rollback, не маскируя исходную ошибку.

## Авторство

Локальные revisions и target commit создаются от имени local authenticated user, который подтвердил apply.

Source actor не становится target `created_by`, потому что:

- source user UUID не переносим;
- source user может отсутствовать на target;
- фактическое действие на target совершил local admin.

Source author information можно сохранить как необязательное, неперсонифицированное provenance summary только если оно уже безопасно присутствует в bundle contract. В первой версии достаточно source commit/release IDs/checksums.

## Идемпотентность и повторы

- Один plan применяется не более одного раза.
- Повторный apply applied plan возвращает conflict с ссылкой на существующий sync run/target commit.
- Первый valid plan может содержать только `unchanged`, если target до sync уже совпадал с source. Его apply всё равно должен сохранить mapping, base bundle и successful sync run, но не создавать пустой workspace commit. Это создаёт common base для следующего three-way comparison.
- Если incoming bundle checksum уже является current base и local state не изменён, новый plan показывает все documents `unchanged`; повторный apply не создаёт новый commit/base и может вернуть `sync_nothing_to_apply` со ссылкой на последний successful run.
- Если новый source release даёт тот же document content, но другое provenance/bundle checksum, apply обновляет acknowledged source provenance/base без пустого workspace commit.
- Параллельные apply в один workspace защищены ETag + workspace lock.
- Network retry выполняется только при создании plan, до target transaction.
- Во время apply сеть не используется: exact bundle уже сохранён в plan.

Не держать database transaction открытой во время remote HTTP request.

## Ограничения HTTP клиента

Remote client обязан иметь:

- connect/TLS/header/body timeouts;
- общий request deadline;
- response body limit до полного чтения;
- проверку `Content-Type`;
- запрет неожиданных compression bomb/неограниченной распаковки;
- ограниченный retry только для idempotent token/export request;
- exponential backoff с малым числом попыток;
- context cancellation;
- закрытие response body;
- no redirects или strict same-origin redirect policy.

Не использовать default `http.Client` без timeout.

## Domain errors

Минимальный набор:

- `sync_remote_not_found`;
- `sync_remote_disabled`;
- `sync_remote_auth_failed`;
- `sync_remote_unavailable`;
- `sync_remote_instance_mismatch`;
- `sync_bundle_too_large`;
- `sync_bundle_schema_unsupported`;
- `sync_bundle_checksum_mismatch`;
- `sync_bundle_invalid`;
- `sync_mapping_invalid`;
- `sync_mapping_requires_source_rewrite`;
- `sync_plan_not_found`;
- `sync_plan_expired`;
- `sync_plan_applied`;
- `sync_plan_conflicted`;
- `sync_target_changed`;
- `sync_missing_integration`;
- `sync_dependency_used_outside_scope`;
- `sync_nothing_to_apply`.

External error body нельзя без фильтрации возвращать local client. Возвращать stable local code и безопасное сообщение; подробности remote status/request ID оставить в server logs.

## Наблюдаемость

Метрики с low-cardinality labels:

- `endge.sync.plan_total{result=valid|conflict|error}`;
- `endge.sync.apply_total{result=succeeded|conflict|failed}`;
- `endge.sync.remote_request_total{operation=token|bundle,result=success|error}`;
- `endge.sync.remote_request_duration_ms{operation=token|bundle}`;
- `endge.sync.bundle_bytes`;
- `endge.sync.documents{operation=create|update|delete|unchanged|conflict}`;
- `endge.sync.active_runs`.

Не использовать remote/workspace/release/document identity в metric labels.

Logs должны содержать correlation fields:

- plan ID;
- sync run ID;
- remote identity;
- source instance/workspace/release;
- target workspace UUID;
- bundle checksum;
- target commit ID;
- counts и duration.

Не логировать token, client secret, source code, полный bundle или target configuration.

## Рекомендуемая структура модулей

- `internal/domain/entities/sync.go` — remote metadata, plan, mapping, run, conflicts;
- `internal/usecase/sync` — target orchestration, comparison и apply;
- `internal/usecase/ports/sync.go` — persistence и remote client ports;
- `internal/platform/syncclient` — OIDC token provider и bounded HTTP client;
- `internal/repo/postgres/sync_repository.go` — plans/mappings/runs/base bundles;
- `internal/api/http/v1/sync` — target API;
- source API переиспользует deployment bundle use case;
- `internal/config` — instance/remotes/timeouts/limits;
- `internal/bootstrap` — Fx wiring;
- новые Goose migrations;
- OpenAPI/README.

Comparison и apply не размещать в HTTP handler или PostgreSQL trigger.

## Конфигурация

Минимальные параметры:

- `INSTANCE_ID`;
- `SYNC_ENABLED=false`;
- `SYNC_PLAN_TTL=30m`;
- `SYNC_REMOTE_REQUEST_TIMEOUT=30s`;
- `SYNC_REMOTE_MAX_BUNDLE_BYTES=16777216`;
- remote metadata/credential refs по выбранному валидируемому формату;
- token refresh skew, например 30s.

Обновить:

- `.env.development.example` без real credentials;
- `.env.production.example`;
- `README.md` с source/target настройкой;
- OpenAPI target/source endpoints;
- deployment notes по TLS, DNS и service account role.

Не помещать пример настоящего token/secret в repository.

## Последовательность реализации

### Фаза A. Remote read boundary

1. Stable instance identity.
2. Remote configuration и secret resolver.
3. OIDC client credentials provider.
4. Bounded source HTTP client.
5. Source export headers/permission.
6. Contract/integration tests remote download.

### Фаза B. Plan и comparison

1. Sync entities/ports/migrations.
2. Mapping validation.
3. Canonical logical document index.
4. Base/incoming/local comparison.
5. Integration/secret/dependency/impact validation.
6. Short-lived persisted plan.
7. Target plan/get endpoints.

### Фаза C. Apply

1. Workspace lock и `If-Match`.
2. Shared document apply lifecycle.
3. Local revisions/mutation batch.
4. `remote_sync` commit.
5. Mapping/base/run update.
6. Idempotency и rollback.
7. Metrics/OpenAPI/docs.

Не начинать apply до готовности plan conflict tests.

## Тестовые сценарии

### Remote client/security

1. Неизвестный remote отклоняется без network request.
2. URL из пользовательского body не принимается.
3. Production HTTP URL отклоняется config validation.
4. Redirect на другой host отклоняется.
5. Token/client secret не появляются в logs/errors.
6. Token кешируется до expiry и безопасно обновляется.
7. Один `401` вызывает один refresh/retry.
8. Timeout/cancellation завершают request.
9. Oversized body прерывается до unbounded allocation.
10. Instance ID mismatch отклоняется.
11. Checksum mismatch отклоняется.

### Plan

1. Первый sync в пустой target создаёт creates.
2. Первый sync с идентичным target даёт unchanged.
3. Первый sync с другим существующим content даёт create collision.
4. Source-only change относительно base даёт update.
5. Local-only change сохраняется, если source unchanged.
6. Одинаковое изменение source/local даёт unchanged.
7. Разные изменения source/local дают `both_changed`.
8. Source delete + unchanged local даёт soft delete.
9. Source delete + changed local даёт conflict.
10. Документ вне previous base никогда не удаляется.
11. Shared dependency delete вне scope запрещён.
12. Missing/incompatible integration делает plan invalid.
13. Secret-bearing incoming content отклоняется.
14. Identity mapping меняет только разрешённые typed fields.
15. Mapping, требующий source rewrite, отклоняется.
16. Plan не меняет target rows/head sequence.

### Apply/transaction

1. Не-admin не создаёт и не применяет plan.
2. Expired/applied plan отклоняется.
3. Неверное confirmation отклоняется.
4. Изменившийся target ETag даёт `412`/stable domain error.
5. Параллельные apply не создают две истории.
6. Mid-transaction failure полностью откатывает documents/revisions/commit/mapping.
7. Успешный apply создаёт локальные revisions с local actor.
8. Создаётся один commit `remote_sync` с точными changes.
9. Source UUID/users не появляются в target FK.
10. Mapping base обновляется только после success.
11. Повторный apply того же plan idempotently отклоняется.
12. Первый no-op apply сохраняет common base без workspace commit.
13. Повторный plan уже acknowledged bundle не создаёт новый base или пустой commit.
14. Во время apply нет remote network request.

## Критерии приёмки

- Target инициирует pull; source не имеет write-доступа к target.
- Синхронизация всегда основана на immutable release/deployment bundle.
- Пользователь не может заставить backend обратиться к произвольному URL.
- Service-to-service credential не хранится/логируется открытым текстом.
- Plan создаётся до любой записи и содержит полный конфликтный summary.
- Без common base первый sync не перезаписывает отличающийся target document.
- `both_changed` не разрешается last-write-wins.
- Удаляются только документы, ранее управляемые тем же mapping, и только soft delete.
- Incoming changes применяются одной транзакцией.
- Target создаёт собственные revisions и commit `remote_sync`.
- Source provenance сохраняется отдельно и не подменяет local author.
- Повторный sync использует сохранённый base для three-way comparison.
- Existing full snapshot import/restore остаётся отдельным сценарием и не переиспользуется как destructive shortcut.

## Риски, которые нельзя скрывать

- Это promotion/synchronization, а не полная Git implementation.
- Автоматический semantic merge source code отсутствует; конфликт требует изменения source/target и нового plan.
- Неполный dependency manifest делает безопасный sync невозможным.
- Shared dependency может влиять на контексты вне selection; без target impact index операция должна fail closed.
- Identity mapping ограничен typed structural fields. Произвольное переписывание source недопустимо.
- OIDC client credentials и network trust должны быть готовы до production запуска.
- Base bundle занимает место в target database; нужна последующая retention/size policy, основанная на реальных объёмах.
