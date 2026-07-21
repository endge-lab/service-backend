# 05. Workspace Context Middleware

## Цель

Создать middleware, который прозрачно разрешает активный workspace из HTTP header и передаёт проверенный scope в handlers/usecases. Задача зависит от `04-workspaces-foundation`.

Backend открытый: middleware не выполняет authentication или authorization и не проверяет пользователей, memberships или роли.

## Контракт

Frontend и другие клиенты передают:

```http
X-Endge-Workspace: default
```

Header содержит `workspace.identity`, не UUID. Поле `workspaceIdentity` в body доменных документов не использовать.

## Логика middleware

Для каждого workspace-scoped запроса:

1. Прочитать и trim `X-Endge-Workspace`.
2. Если header отсутствует — вернуть `400 workspace_required`.
3. Вызвать `GetByIdentity(workspaceIdentity)` из workspace usecase.
4. Если workspace не найден — вернуть `404 workspace_not_found`.
5. Положить разрешённый scope в стандартный Go `context.Context` и вызвать следующий handler.

Создать transport-independent context value и accessor, например:

```go
type WorkspaceScope struct {
    ID       uuid.UUID
    Identity string
}

func WithWorkspace(ctx context.Context, scope WorkspaceScope) context.Context
func WorkspaceFromContext(ctx context.Context) (WorkspaceScope, bool)
```

Accessor не должен находиться в package конкретного handler и не должен зависеть от Fiber. Handlers/usecases не читают header повторно и не резолвят workspace самостоятельно.

## Подключение

Middleware применяется только к workspace-scoped domain routes. Зависимости от authentication middleware быть не должно.

Не применять его к:

```text
/health
/swagger
/swagger/openapi3.yaml
/api/v1/workspaces
```

Добавить `X-Endge-Workspace` в CORS `AllowHeaders` и описать header в OpenAPI для scoped operations.

Repository queries должны использовать только проверенный `WorkspaceScope.ID`. Raw header нельзя передавать напрямую в SQL.

## Errors

Использовать общий формат сервиса:

```json
{
  "code": "workspace_required",
  "message": "Workspace context is required",
  "details": {}
}
```

Коды:

```text
workspace_required
workspace_not_found
```

Не логировать содержимое workspace config, `sse.manualToken` и другие secrets. В logs/traces допустим только проверенный `workspace.id` и `workspace.identity`.

## Tests и Acceptance Criteria

Проверить сценарии: header отсутствует; identity пустой; workspace не найден; workspace найден; scope доступен следующему handler; unscoped routes работают без header; CORS preflight разрешает header.

Middleware считается готовым, когда feature handlers используют `WorkspaceFromContext`, а ни один workspace-scoped repository не выполняет запрос без `workspaceID`.
