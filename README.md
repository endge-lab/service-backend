# Endge Backend

Backend визуального конфигуратора Endge. Сервис хранит source-first документы,
workspace-историю и portable releases; frontend подключается к API через
отдельный adapter.

## Архитектура

Запрос проходит через authentication, current-user projection и workspace
authorization. HTTP handlers вызывают usecase, usecase управляет транзакцией и
работает с PostgreSQL только через ports/repositories.

Каждый HTTP resource владеет своим адаптером:

```text
internal/api/http/v1/<resource>/
├── handler.go
├── routes.go
├── transport.go
└── usecase.go
```

`shared` содержит только общие HTTP-примитивы и stateless operation-функции:
strict decoding, декларативную validation, ETag/`If-Match`, list filters и
единое формирование ответа. Resource DTO, маршруты, usecase-порты и stateful
handlers в `shared` не размещаются.

Request DTO используют `validate`-теги из `service-kit-go`. Неизвестные JSON-поля
отклоняются, а ошибки полей возвращаются единообразно как
`400 validation_error` в `details.fields`. HTTP handlers зависят от локального
`UseCase` interface; concrete application use cases подключаются только в
bootstrap. В `internal/usecase/<resource>` реализация называется `UseCase`,
общий document-механизм — `documents.Lifecycle`, а координация полного состояния
workspace — `workspace_state.Coordinator`. Голое имя `Service` для application
use case не используется.

Согласованные коллекции:

```text
projects, tenants, environments, folders,
types, queries, data-views, compositions,
stores, streams, updates, mocks, components,
actions, filters, converters, computations,
vocabs, i18n-bundles, auth-profiles, navigations, styles
```

`parameters`, `pages`, `page-templates`, `policies`, `versions`, legacy
components, hard-delete, release tags и channels намеренно отсутствуют в MVP.

## Локальный запуск

```bash
cp .env.development.example .env.development
make migrate-up
make run
```

Development разрешает явно включённый фиксированный actor:

```env
APP_ENV=development
AUTH_MODE=dev
AUTH_DEV_SUBJECT=developer
AUTH_DEV_USERNAME=developer
AUTH_DEV_DISPLAY_NAME=Local Developer
AUTH_DEV_PLATFORM_ADMIN=true
```

`AUTH_MODE=dev` запрещён при `APP_ENV=production`.

## Версия backend

Единственный источник build metadata сервиса — корневой файл `VERSION`:

```text
APP_VERSION=0.10.0
WORKSPACE_SCHEMA_VERSION=1
```

Перед каждым изменением backend повышается `APP_VERSION` по SemVer.
`WORKSPACE_SCHEMA_VERSION` повышается только при несовместимом изменении полного
workspace export/import contract. Локальная, Docker- и удалённая сборки вшивают
оба значения в бинарник; Kubernetes их не переопределяет. Публичный
`GET /version` возвращает оба значения, а `GET /health` — версию application.
`GET /version` дополнительно запрашивает metadata
подключённых backend-сервисов через их штатный transport и возвращает массив
`services`. Недоступный сервис остаётся в массиве со статусом `unavailable`, но
без внутренних адресов и infrastructure error; это не меняет liveness backend.
Результат кешируется на `AI_WORKBENCH_HEALTH_CACHE_TTL`.

## Production OIDC

Production запускается только с валидной OIDC/JWKS-конфигурацией:

```env
APP_ENV=production
AUTH_MODE=oidc
AUTH_PROVIDER_ID=primary
AUTH_ISSUER=https://keycloak.example/realms/endge
AUTH_JWKS_URL=https://keycloak.example/realms/endge/protocol/openid-connect/certs
AUTH_ALLOWED_AUDIENCES=endge-configurator
AUTH_ALLOWED_ALGORITHMS=RS256
AUTH_USERNAME_CLAIM=preferred_username
AUTH_DISPLAY_NAME_CLAIM=name
AUTH_GROUPS_CLAIM=groups
AUTH_PLATFORM_ADMIN_GROUPS=endge-platform-admins
AUTH_LOGIN_ADAPTER=oidc
AUTH_AUTHORIZATION_URL=https://keycloak.example/realms/endge/protocol/openid-connect/auth
AUTH_TOKEN_URL=https://keycloak.example/realms/endge/protocol/openid-connect/token
AUTH_LOGOUT_URL=https://keycloak.example/realms/endge/protocol/openid-connect/logout
AUTH_CLIENT_ID=endge-configurator
AUTH_REDIRECT_URL=https://backend.example.com/auth/callback
AUTH_RETURN_URL=https://configurator.example.com
AUTH_ALLOWED_RETURN_ORIGINS=https://configurator.example.com
AUTH_SESSION_COOKIE_NAME=endge_configurator_session
ENCRYPTION_KEY_ID=v1
ENCRYPTION_KEY=<base64-encoded-32-byte-key>
ENCRYPTION_PREVIOUS_KEYS=
AUTH_SESSION_CLEANUP_INTERVAL=15m
AUTH_COOKIE_SECURE=true
```

Keycloak-специфического кода нет: любой совместимый OIDC provider подключается
этими переменными. Первый валидный запрос создаёт локальную проекцию пользователя
в `service_users`; следующие запросы сохраняют тот же UUID и обновляют публичные
имя/username.

Browser flow не раскрывает provider tokens Конфигуратору:

```text
GET  /auth/login
GET  /auth/callback
GET  /auth/session
POST /auth/logout
```

Configurator передаёт текущий browser URL через `returnTo`. Backend возвращает
пользователя на него только при совпадении origin с `AUTH_RETURN_URL` или одним
из comma-separated значений `AUTH_ALLOWED_RETURN_ORIGINS`; иначе используется
`AUTH_RETURN_URL`. Wildcard не поддерживается, чтобы backend не становился open
redirect. Keycloak всегда возвращает браузер только на `AUTH_REDIRECT_URL`.

Callback хранит в PostgreSQL только зашифрованный refresh token (если provider
его выдал) и выдаёт браузеру opaque `HttpOnly` cookie. Access token после
проверки не сохраняется. `GET /health` публичен. Весь `/api`
принимает эту cookie или bearer token, кроме development с явным `AUTH_MODE=dev`.

Зашифрованные auth-значения содержат версию ключа. Для ротации сначала задайте
новые `ENCRYPTION_KEY_ID` и `ENCRYPTION_KEY`, а старый
ключ перенесите в список вида
`ENCRYPTION_PREVIOUS_KEYS=v1:<base64>,v0:<base64>`. Новые и
обновляемые sessions шифруются текущим ключом; предыдущие используются только
для чтения. Старые ключи можно убрать после истечения большего из
`AUTH_SESSION_TTL` и `AUTH_TRANSACTION_TTL` для данных, созданных до ротации. Просроченные login
transactions и sessions удаляются фоновым процессом с интервалом
`AUTH_SESSION_CLEANUP_INTERVAL`, а не внутри login request.

Таблицы `configurator_auth_transactions` и `configurator_auth_sessions` являются
внутренним security-state сервиса и не входят в workspace snapshot, export,
backup или release.

## Workspace и конкурентная запись

Workspace выбирается заголовком:

```http
X-Endge-Workspace: default
```

Активный пользователь имеет implicit `editor` в `default`. Для остальных
workspace нужен membership `viewer`, `editor` или `admin`. Явный membership в
`default` переопределяет implicit роль.

PATCH, DELETE и restore требуют ETag предыдущего ответа:

```http
If-Match: "3"
```

Без заголовка API отвечает `428 precondition_required`, при устаревшей revision —
`409 revision_conflict`. DELETE всегда выполняет soft-delete.

## История

- Каждая фактическая запись создаёт полный JSON snapshot в `document_revisions`.
- Одно серверное действие связывает затронутые документы одним `mutation_batch`.
- Commit фиксирует pending revisions всего workspace с policy `preserve` или
  `squash`.
- Release является immutable portable snapshot существующего Commit и скачивается
  без replay истории.
- Restore revision/commit/release добавляет новую историю и не переписывает старую.
- Export/import не содержит UUID-связей, пользователей, memberships, истории и
  секретов.

## Snapshot, перенос и backups

`GET /api/v1/domain` возвращает live workspace со всеми документами и
server-only полем `state`. `GET /api/v1/domain/export` отдаёт тот же переносимый
контракт без локальных UUID, истории и времён хранения. Оба export endpoint
возвращают JSON inline; `?download=true` включает скачивание файла.

Перед export и проверкой import backend приводит переносимую часть домена к одной
идемпотентной канонической форме. Текущий `domainVersion` использует контракт
`dv2:sha256`; поэтому export/import/export одного snapshot сохраняет одинаковый
hash независимо от legacy-представления Action, Vocab и совместимых defaults.
Snapshot с `dv1:sha256` по-прежнему проверяется старым алгоритмом, но после
успешного import сохраняется уже с актуальным `dv2`.

Поле `schemaVersion` полного snapshot должно точно совпадать со встроенным
`WORKSPACE_SCHEMA_VERSION`. При несовпадении backend возвращает
`workspace_schema_unsupported`; файл нужно заново экспортировать из обновлённого
исходного backend. Frontend не мигрирует и не формирует этот контракт.

Безопасный импорт полного domain snapshot выполняется в два шага:

```text
POST /api/v1/domain/import/plan
POST /api/v1/domain/import
```

Второй запрос требует `planId`, подтверждение identity текущего workspace и
`If-Match` из плана. В одной транзакции backend блокирует workspace, создаёт новые
document revisions и один commit с `operation=import`. Новые документы создаются,
совпавшие по `(type, identity)` полностью заменяются новой revision, отсутствующие
в полном snapshot получают soft-delete revision. Предыдущие revisions, commits,
releases, пользователи и memberships не удаляются; состояние до импорта можно
восстановить через restore родительского commit.

Backups доступны через `/api/v1/domain/backups`: manual backup принимает
опциональное описание, `last` выбирает последнюю копию, `/archive` возвращает ZIP
всех доступных snapshots. Manual backups бессрочны; ранее созданные `pre_import`
backups остаются доступными до истечения своего срока хранения.

Последний release экспортируется через
`GET /api/v1/releases/last/export`; по умолчанию это JSON, а
`?download=true` возвращает attachment.

Release artifact JSON лениво кэшируется в памяти backend-процесса. Перед чтением
artifact API всегда проверяет metadata и checksum, поэтому `ETag` / `If-None-Match`
безопасно возвращают `304` после проверки авторизации workspace. Лимиты задают
`RELEASE_ARTIFACT_CACHE_ENABLED`, `RELEASE_ARTIFACT_CACHE_MAX_BYTES` (по умолчанию
64 MiB) и `RELEASE_ARTIFACT_CACHE_MAX_ITEM_BYTES` (16 MiB). Каждая реплика имеет
свой локальный LRU-кэш: cache miss на другой реплике может один раз прочитать JSON
из PostgreSQL, но не влияет на корректность или свежесть `last`.

Полный HTTP-контракт генерируется из Swagger-аннотаций handler-методов и
transport DTO командой `make docs`. Результат сохраняется в
[`docs/openapi3.yaml`](docs/openapi3.yaml) и встраивается в binary. В development Scalar
доступен на `/swagger`.

## Миграции и проверки

```bash
make migrate-up
make migrate-down
make sqlc
```

Полная стратегия тестирования, Docker-наборы и защитный механизм тестовой БД
описаны в [`docs/Тестирование.md`](docs/%D0%A2%D0%B5%D1%81%D1%82%D0%B8%D1%80%D0%BE%D0%B2%D0%B0%D0%BD%D0%B8%D0%B5.md).

Недокерные проверки:

```bash
make test-unit
```

Интеграционные и E2E-наборы сами создают PostgreSQL 17 через Testcontainers.
Внешний DSN намеренно не поддерживается:

```bash
make test-integration
make test-e2e
make test-critical
```
