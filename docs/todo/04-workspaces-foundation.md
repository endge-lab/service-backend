# 04. Workspaces Foundation — progress

Статус: в работе.

## Выполнено

### Пункт 1 — анализ задания

- Сверены задание, текущая схема, SQLC-контракты и структура слоёв.
- Выявлены все реализованные workspace-scoped таблицы и зависимость старых
  repository от появления `workspace_id`.
- Согласовано: workspace scope существующих API будет передаваться через
  request context; позднее задача 05 централизует это middleware.

### Пункт 2 — таблицы, SQLC и доменный контракт

- Реализована таблица `workspaces` с полной nested `configuration JSONB`.
- В 13 реализованных workspace-scoped таблиц добавлен `workspace_id NOT NULL`.
- Identity uniqueness перенесена в scope workspace.
- Добавлены Workspace SQLC CRUD-запросы и generated-код.
- Добавлены `RWorkspace`, `EndgeConfiguration`, system default и
  `WorkspacesRepository` port.
- Миграции успешно применены на изолированной чистой PostgreSQL-БД; временная
  тестовая БД затем удалена.

### Пункт 3 — PostgreSQL repository

- Реализован `WorkspacesRepository` с `Create`, `List`, `GetByIdentity` и
  `Update`; repository подключён в Fx.
- Добавлены отдельные JSONB mappers Workspace с явной обработкой ошибок
  сериализации и десериализации.
- `sse.manualToken` не добавляется в repository-логи и trace fields.
- Существующие mappers и SQLC create-контракты workspace-scoped сущностей
  передают `workspace_id`; Project identity-операции берут scope из context.
- Добавлен типизированный helper для передачи разрешённого workspace UUID через
  `context.Context`; handler подключит его на последующем HTTP-пункте.
- Проверки успешно пройдены: `go test ./internal/repo/postgres`, `go test ./...`,
  `go vet ./...`, `git diff --check`.

### Пункт 4 — usecase/service слой

- Реализован `Workspace` usecase с `Create`, `List`, `GetByIdentity`, `Update`.
- Create подставляет system default, если configuration не передана.
- Валидация проверяет arrays, uniqueness, default locale/theme/adapter и правила
  optional SSE; configuration при update заменяется целиком без JSON merge.
- Workspace usecase подключён в Fx.
- Project Create требует resolved workspace scope из context и передаёт его в
  Project и создаваемые system root folders.
- Публичные Workspace repository и usecase-методы документируют полный flow:
  вход, валидацию/маппинг, действия, результат и доменные ошибки.
- Обновлены тесты Project Create для явной проверки workspace scope.
- Проверки успешно пройдены: `go test ./...`, `go vet ./...`, `git diff --check`.

### Пункт 5 — HTTP handler и workspace scope

Статус: завершён.

#### Уже сделано

- Создан `internal/api/http/v1/workspace` с HTTP-контрактом, handler, DTO и
  routes.
- Добавлены четыре endpoint: list, create, get by identity и patch.
- Handler использует bind, HTTP validation, mapping, usecase и
  `respond.RespondDomainError`; бизнес-логика в него не добавлена.
- Обычные Workspace responses redacted: `sse.manualToken` не возвращается.
- Handler и маршруты подключены в Fx и `SetupRoutes`.
- Проверка bootstrap dependency graph проходит.

#### Завершение после возврата к пункту 5

1. Выполнено: реализован минимальный workspace-context middleware:
   - прочитать `X-Endge-Workspace`;
   - разрешить workspace identity в UUID;
   - передать UUID далее через `entities.WithWorkspaceID`;
   - не логировать configuration и secret-поля.

2. Выполнено: middleware подключён к workspace-scoped routes: Projects,
   Folders, Components Legacy, Converters, Queries и Data Views. Workspace API
   не должен требовать этот header.

3. Выполнено: `docs/openapi3.yaml` содержит Workspace schemas: configuration,
   locale, theme, SSE, create, patch, response и list; в описании endpoint
   указана redaction `manualToken`.

4. Выполнено: добавлены подробные Swagger-комментарии над Workspace handler methods и
   документацию flow handler-слоя.

5. Выполнено: добавлены middleware и handler tests; выполнены
   `go test ./...`, `go vet ./...`, `git diff --check`.

### Пункт 6 — tests по acceptance criteria

Статус: завершён.

- Create без configuration проверен на system default locales ru/en, themes
  light/dark и default locale/theme.
- Проверены configuration invariant для default theme и полная replacement
  configuration при update без JSON merge.
- Проверен JSONB round-trip полной configuration и явная ошибка malformed JSONB.
- Проверены обязательность `X-Endge-Workspace`, передача resolved UUID в context
  и redaction `sse.manualToken` в HTTP response.
- Добавлены HTTP CRUD tests Workspace для `POST`, `GET list`, `GET by identity`
  и `PATCH`; они проверяют transport input и отсутствие secret в каждом обычном
  response. Некорректный create request возвращает `validation_error` и HTTP 400.
- Добавлена table-driven проверка uniqueness vars/locales/themes/adapters,
  fallback locale, direction, default adapter и допустимых SSE auth modes.
- Выполнен tagged integration test `WorkspacesRepository` на чистой временной
  PostgreSQL-БД: накатывание всех миграций, `Create → Get → Update → List` и
  конфликт duplicate identity. Временные container, volume и network удалены
  после проверки.
- Проверки успешно пройдены: `go test ./...`, `go vet ./...`, `git diff --check`.
