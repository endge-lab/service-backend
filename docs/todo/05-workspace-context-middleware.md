# 05. Workspace Context Middleware — история изменений

## Статус

Выполнено. Исходная постановка задачи находится в
`../task/05-workspace-context-middleware(DONE).md`.

## Реализовано

### Context scope

В `internal/domain/entities/workspace.go` добавлены transport-independent
`WorkspaceScope`, `WithWorkspace` и `WorkspaceFromContext`. Для совместимости
оставлены `WithWorkspaceID` и `WorkspaceIDFromContext`.

```go
type WorkspaceScope struct {
    ID       uuid.UUID
    Identity string
}
```

### Middleware и Fx

- Создан `WorkspaceContextMiddleware` с конструктором
  `NewWorkspaceContextMiddleware`.
- Middleware trim'ит `X-Endge-Workspace`, вызывает injected `WorkspaceResolver`
  и записывает разрешённые ID/identity в `c.UserContext()`.
- Пустой header возвращает `400 workspace_required`.
- Ошибка `not_found` от resolver преобразуется в `404 workspace_not_found`.
- Глобальный resolver и mutex удалены.
- Provider middleware вынесен в `internal/bootstrap/workspace_context.go` и
  подключён через Fx; зависимость передаётся в HTTP handler constructor.

### Маршруты

Middleware включён только для scoped групп:

- projects;
- folders;
- components-legacy;
- converters;
- queries;
- data-views.

`/health`, `/swagger`, `/swagger/openapi3.yaml` и `/api/v1/workspaces` остались
без workspace header. Authentication, memberships и roles в рамках задачи не
добавлялись.

### Application и persistence boundary

Handlers передают в use case обычный `c.UserContext()` и не прокидывают Fiber
context. Header не читается повторно и не передаётся в body или SQL.

- Project create читает workspace ID из context, поскольку он нужен новой
  сущности.
- Дочерние сущности получают `workspace_id` от разрешённого project.
- Update сохраняет исходный `workspace_id`.
- Repository boundary требует scope в context и отклоняет entity с workspace,
  отличающимся от request scope.

Все SQL-запросы projects, folders, components-legacy, converters, queries и
data views получают фильтр/параметр `workspace_id`. Raw header в SQL не попадает.

### HTTP contract и OpenAPI

- `X-Endge-Workspace` добавлен в CORS `AllowHeaders`.
- Во всех 42 scoped OpenAPI операциях header описан как обязательный.
- Root workspace endpoints header не требуют.
- Middleware не логирует workspace configuration, `sse.manualToken` или другие
  secrets; в scope присутствуют только ID и identity.

## Проверки

Добавлены и проходят:

- middleware tests: отсутствующий/пустой header, trim identity,
  `workspace_not_found`, scope в следующем handler;
- HTTP-level tests: unscoped routes без header и CORS preflight;
- OpenAPI test: каждый scoped operation требует `X-Endge-Workspace`;
- repository unit tests: scope обязателен, entity scope совпадает с request
  scope;
- PostgreSQL integration test: workspace B не читает и не обновляет project
  workspace A.

Выполненные проверки:

```bash
GOCACHE=/tmp/service-backend-go-build go test ./...
go test -tags=integration ./internal/repo/postgres
git diff --check
```

Интеграционный тест запускался на отдельной чистой PostgreSQL. Временная БД после
проверки удалена.

## Итоговый flow

```text
X-Endge-Workspace
  -> WorkspaceContextMiddleware
  -> context.Context (WorkspaceScope)
  -> handler передаёт UserContext
  -> use case
  -> scoped repository / SQL workspace_id filter
```

Feature handlers не извлекают scope напрямую: они передают `c.UserContext()` в
application layer. `WorkspaceFromContext` используется там, где scope нужен для
создания сущности или repository-проверки. Это сохраняет handlers независимыми
от domain context и не допускает повторного чтения header.
