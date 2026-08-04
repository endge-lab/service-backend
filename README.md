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
AUTH_SESSION_COOKIE_NAME=endge_configurator_session
AUTH_SESSION_ENCRYPTION_KEY=<base64-encoded-32-byte-key>
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

Callback хранит provider tokens в зашифрованной PostgreSQL session и выдаёт
браузеру только opaque `HttpOnly` cookie. `GET /health` публичен. Весь `/api`
принимает эту cookie или bearer token, кроме development с явным `AUTH_MODE=dev`.

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

Полная замена domain state выполняется в два шага:

```text
POST /api/v1/domain/import/plan
POST /api/v1/domain/import
```

Второй запрос требует `planId`, подтверждение identity текущего workspace и
`If-Match` из плана. В одной транзакции backend блокирует workspace, создаёт
`pre_import` backup, очищает документы и историю, импортирует snapshot и создаёт
начальный commit. Пользователи, memberships, сам workspace и глобальный catalog
интеграций не удаляются.

Backups доступны через `/api/v1/domain/backups`: manual backup принимает
опциональное описание, `last` выбирает последнюю копию, `/archive` возвращает ZIP
всех доступных snapshots. Срок хранения автоматических `pre_import` backups
задаётся `IMPORT_BACKUP_RETENTION_DAYS`; manual backups бессрочны.

Последний release экспортируется через
`GET /api/v1/releases/last/export`; по умолчанию это JSON, а
`?download=true` возвращает attachment.

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
