# Карта E2E покрытия API

## Назначение

Документ фиксирует исходное покрытие HTTP API перед рефакторингом
PostgreSQL document adapters. Его задача — сделать регрессию проверяемой:
одинаковый набор тестов должен быть зелёным до и после изменения внутренней
реализации репозиториев.

## Что считается E2E в этом проекте

Тесты с тегом `e2e` поднимают временный PostgreSQL-контейнер, применяют
миграции и создают настоящее Fiber-приложение. Запрос проходит через роутинг,
middleware, аутентификацию, handler, use case и PostgreSQL.

Вызов выполняется через `app.Test`, поэтому это не проверка Docker Compose,
внешней сети, браузера или Scalar. В частности, внешние зависимости AI
Workbench и настоящий OIDC-провайдер должны эмулироваться тестовым double.

## Состояние карты

Источник операций: `docs/openapi3.yaml`.

- Всего OpenAPI операций: **210**.
- Покрыто E2E: **210** (**100%**).
- Не покрыто E2E: **0**.

## Покрытые операции

### Документные коллекции: 138 операций

`TestAllDocumentHTTPContracts` в `test/e2e/http_test.go` проверяет для каждой
коллекции одинаковый контракт:

- `GET /api/v1/{collection}`;
- `POST /api/v1/{collection}`;
- `GET /api/v1/{collection}/{identity}`;
- `PATCH /api/v1/{collection}/{identity}` с обязательным `If-Match`;
- `DELETE /api/v1/{collection}/{identity}`;
- `POST /api/v1/{collection}/{identity}/restore`.

Покрыты все 23 коллекции:

`actions`, `auth-profiles`, `components`, `compositions`, `computations`,
`configurations`, `converters`, `data-views`, `environments`, `filters`,
`folders`, `i18n-bundles`, `mocks`, `navigations`, `projects`, `queries`,
`stores`, `streams`, `styles`, `tenants`, `types`, `updates`, `vocabs`.

Дополнительно этот тест закрепляет ETag, optimistic locking, soft delete,
`includeDeleted`, валидацию JSON и CSRF-защиту cookie-аутентификации.

### Прочие покрытые группы

| Группа | Проверяемые операции и свойства | Тест |
|---|---|---|
| Session, health и browser auth | `GET /api/session/me`, `GET /health`, `/auth/login`, callback, logout, bearer/OIDC, синхронизация локального пользователя | `http_test.go`, `configurator_auth_test.go` |
| Folder safety | immutable roots, циклическая вложенность, перенос содержимого при удалении | `http_test.go` |
| Bulk move | `POST /api/v1/domain/documents/move`, атомарность и conflict | `http_test.go` |
| Commits | list, get, diff, plan, create, restore plan, restore; матрица ролей | `history_test.go`, `rbac_test.go` |
| Domain и snapshots | domain, export, status, import plan/import, backup list/create/get/export/archive | `history_test.go` |
| Revisions | список revisions и restore revision | `history_test.go` |
| Revision detail | detail конкретной revision, 401/403/404 и workspace isolation | `revision_detail_test.go` |
| Releases | create, list, get, export, conditional `304`, restore plan, restore, alias `last`, workspace isolation | `history_test.go`, `release_read_test.go` |
| Service API | public `GET /version`; поиск service users с pagination и RBAC | `service_api_test.go` |
| Workspaces | create, list, get, patch с ETag, list members, membership mutations, role matrix и isolation | `service_api_test.go`, `rbac_test.go` |
| Integrations | create, list, get, patch, delete, restore, soft-delete и ETag/RBAC | `service_api_test.go`, `rbac_test.go` |
| Backend connections | list, create, delete, нормализация URL и platform-admin RBAC | `service_api_test.go` |
| Access grants | list, put, delete, bulk `selected`/`all-active`, RBAC и реальное изменение доступа после grant/revoke | `access_grants_test.go` |
| AI catalog | adapters, connections и model profiles: полный lifecycle, RBAC и encrypted credentials | `ai_api_test.go` |
| AI assistant | capabilities, conversations, messages, reset и SSE run через реальный gRPC adapter + fake Workbench | `ai_api_test.go` |

## Непокрытые операции

Отсутствуют: все 210 операций из `docs/openapi3.yaml` проходят как минимум
один E2E-сценарий через настоящее Fiber-приложение и временный PostgreSQL.

## Смежные тестовые уровни

Integration-тесты используют реальный PostgreSQL, но обходят HTTP transport:

- `test/integration/documents_test.go` — lifecycle всех document repositories,
  rollback транзакций, commit/release/restore и cache artifact reader;
- `test/integration/migrations_test.go` — миграции, bootstrap и legacy Vocab;
- `test/integration/auth_sessions_test.go` — session cleanup и refresh locking.

Unit-тесты проверяют transport mapping, config, auth, validation, cache reader и
другие изолированные компоненты. Они полезны, но не заменяют проверку маршрута
через полный HTTP pipeline.

## Как использовать для точки 0

Перед рефакторингом и после него нужно запускать:

```bash
make test-critical
go test ./...
```

`make test-critical` включает unit, integration и E2E. Для integration/E2E
нужен доступ к Docker, потому что `TestMain` создаёт временный PostgreSQL.

Полная точка 0 на все 210 операций достигнута. Рефакторинг document
repositories можно начинать только при зелёных проверках из этого документа.
