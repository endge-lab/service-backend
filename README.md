# Endge Service Template Go

`service-template-go` — пример Go-сервиса, который показывает, как подключать `github.com/endge-lab/service-kit-go` и строить backend с HTTP API, PostgreSQL, миграциями и чистыми слоями.

Шаблон по умолчанию подходит для простого монолита/API:

- HTTP на Fiber;
- PostgreSQL через `pgx`;
- миграции через `goose`;
- DI через `fx`;
- health/version endpoints;
- OpenAPI-файл в `docs/openapi.yaml`;
- пример фичи `Todo`;
- auth выключен по умолчанию;
- telemetry выключена по умолчанию;
- Redpanda/Kafka выключены по умолчанию.

## Как использовать

1. Скопируйте репозиторий или создайте новый сервис на его основе.
2. Замените module path в `go.mod`:

   ```go
   module github.com/your-org/your-service
   ```

3. Замените импорты `github.com/endge-lab/service-template-go/internal/...` на module path нового сервиса.
4. Скопируйте env-файл:

   ```bash
   cp .env.development.example .env.development
   ```

5. Настройте `DATABASE_URI`, `APP_NAME`, `PUBLIC_URL`, `CORS_ALLOWED_ORIGINS`.
6. Запустите тесты:

   ```bash
   go test ./...
   ```

7. Запустите сервис:

   ```bash
   make run
   ```

## Локальная разработка с service-kit-go

Для разработки рядом с локальной версией kit используйте `go.work`. Это аналог workspace в frontend-проектах.

Пример структуры:

```text
workspace/
├── service-kit-go/
└── service-template-go/
```

Команда:

```bash
cd workspace
go work init ./service-kit-go ./service-template-go
```

В `service-template-go/go.mod` при этом остаётся обычная published-зависимость:

```go
require github.com/endge-lab/service-kit-go v0.1.0
```

Локально Go подставит папку `./service-kit-go`, а без `go.work` скачает tagged-версию из GitHub.

До публикации первого тега `github.com/endge-lab/service-kit-go@v0.1.0` команды `go mod tidy` и `go test` в template могут пытаться скачать ещё несуществующую версию. Для bootstrap-проверки можно временно добавить:

```go
replace github.com/endge-lab/service-kit-go => ../service-kit-go
```

Этот `replace` не нужно коммитить в публичный template. После публикации тега он больше не нужен.

## Auth

Auth опционален. По умолчанию:

```env
AUTH_ENABLED=false
```

В этом режиме `/api/todos` открыт, а `/api/session/me` не регистрируется.

Чтобы включить JWT/JWKS auth:

```env
AUTH_ENABLED=true
AUTH_SERVICE_URL=https://auth.example.com
AUTH_ISSUER=https://auth.example.com
AUTH_ALLOWED_AUDIENCES=your-audience
```

## Redpanda/Kafka

Redpanda опциональна. По умолчанию:

```env
REDPANDA_ENABLED=false
```

Включайте её только в event-driven сервисах:

```env
REDPANDA_ENABLED=true
REDPANDA_BROKERS=redpanda:9092
REDPANDA_CLIENT_ID=your-service
```

## Telemetry

Telemetry опциональна. По умолчанию:

```env
TELEMETRY_ENABLED=false
OTEL_EXPORTER_OTLP_ENDPOINT=
```

В этом режиме сервис не подключается к OTLP collector. Если нужен OpenTelemetry export:

```env
TELEMETRY_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
OTEL_EXPORTER_OTLP_INSECURE=true
```

## Logging

Шаблон использует `service-kit-go/logging`, но это не обязательное правило для всех сервисов. В собственном сервисе можно:

- оставить kit-логгер для JSON logs;
- заменить на стандартный `log/slog`;
- передавать noop logger там, где логи не нужны.

## Публикация

Template и kit публикуются как обычные GitHub-репозитории. Версия kit становится доступной для `go get` после тега:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Обновление template на новую версию kit:

```bash
go get github.com/endge-lab/service-kit-go@v0.1.0
go mod tidy
```

## Проверки

```bash
go test ./...
docker compose --env-file .env.development config
```

Для проверки поведения без локального workspace:

```bash
GOWORK=off go test ./...
```

Эта команда начнёт работать после публикации `github.com/endge-lab/service-kit-go` с версией, указанной в `go.mod`.
