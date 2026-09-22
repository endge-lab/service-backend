# Presence активных пользователей Configurator внутри workspace

## Статус и приоритет

- Статус: требуется реализация после согласования режима deployment.
- Приоритет: низкий/средний; это UX-функция, а не защита данных.
- Область изменений: backend и публичный WebSocket contract. Реализация UI не входит в эту задачу.
- Ключевое ограничение первой версии: один экземпляр backend обслуживает presence целиком.

## Бизнес-контекст

Configurator — браузерное приложение, в котором пользователи редактируют содержимое workspace. Пользователю нужно видеть, работает ли кто-то ещё в том же workspace прямо сейчас.

Первая версия отвечает только на вопрос: **сколько уникальных пользователей сейчас держат открытое активное соединение с этим workspace**.

Это не совместное редактирование. Presence не блокирует документы, не объединяет изменения и не заменяет optimistic concurrency через revision/`If-Match`.

## Термины

- **Workspace** — изолированная область конфигурации и RBAC. Presence всегда считается внутри одного workspace.
- **User** — `service_users.id`, полученный из проверенного server-side auth context.
- **Connection** — одна WebSocket-связь одной вкладки/окна браузера.
- **Unique user count** — количество разных user ID, у которых есть хотя бы одно живое connection в workspace.
- **Connection count** — техническое число соединений. Оно может быть больше user count из-за нескольких вкладок.
- **Presence hub** — один backend-компонент, владеющий всеми live connections и агрегированным состоянием.

## Текущее состояние backend

Основные точки:

- `internal/api/http/middleware/auth.go` аутентифицирует bearer token или Configurator cookie;
- `internal/api/http/middleware/current_user.go` разрешает server user;
- `internal/api/http/v1/workspace/middleware.go` проверяет `X-Endge-Workspace` и RBAC;
- `internal/usecase/workspaces/queries.go` содержит authoritative `Authorize`;
- `internal/api/http/setup_routes.go` собирает route chain;
- `internal/domain/entities/session.go` и Configurator auth sessions описывают login session, но не присутствие пользователя онлайн.

Таблица browser sessions не подходит для presence:

- session живёт дольше вкладки;
- пользователь может закрыть браузер, но session останется валидной;
- одна session не означает активную работу;
- у пользователя может быть несколько вкладок или устройств.

## Цель

После реализации авторизованный пользователь может открыть WebSocket для конкретного доступного ему workspace и получать:

- первоначальный snapshot количества пользователей;
- обновление количества после подключения другого пользователя;
- обновление количества после ухода последнего connection пользователя;
- корректное удаление оборванных соединений по heartbeat timeout.

## Не входит в задачу

- список имён, email или аватаров пользователей;
- документ-level presence;
- курсоры, выделения и совместное редактирование;
- блокировки документов;
- автоматическое разрешение revision conflicts;
- чат;
- хранение presence в PostgreSQL;
- Kafka/Redpanda для первой версии;
- корректная агрегация между несколькими репликами backend.

## Почему нужен отдельный route

Browser WebSocket API не позволяет установить произвольный HTTP header `X-Endge-Workspace`. Поэтому существующий scoped group с `RequireWorkspace()` нельзя применить без изменения transport contract.

Использовать явный route:

```text
GET /api/v1/workspaces/{workspaceIdentity}/presence/ws
```

Route должен находиться под общими `AuthMiddleware` и `CurrentUser.Resolve`, но до group, который требует `X-Endge-Workspace`.

Handler обязан взять `workspaceIdentity` только из path и вызвать тот же `workspaces.UseCase.Authorize`, который используется обычным workspace middleware. Нельзя считать наличие валидной cookie достаточным правом на любой workspace.

## Безопасность WebSocket upgrade

### Authentication

- Configurator browser использует существующую HttpOnly session cookie.
- Bearer-auth клиент может использовать WebSocket только если выбранная WebSocket library/клиент действительно позволяет передать `Authorization` во время upgrade. Не добавлять token в query string.
- Query string не должен содержать access token, session token или другой credential.
- Истёкшая/отозванная cookie возвращает `401` до upgrade.
- Auth middleware должен явно сохранить в request context способ аутентификации (`cookie` или `bearer`). Presence handler не должен угадывать его по случайному наличию заголовков/cookie после факта.

### Origin

WebSocket handshake использует HTTP GET. Текущая cookie CSRF-проверка выполняется только для unsafe methods, поэтому WebSocket требует отдельной обязательной проверки `Origin` до upgrade.

Правила:

- `Origin` обязателен для cookie-auth browser connection;
- origin проверяется по `HTTP.CORSAllowedOrigins` тем же алгоритмом, что обычный CORS/CSRF;
- общую функцию проверки origin нужно переиспользовать, а не копировать с отличающимися правилами;
- запрещённый или пустой browser origin получает `403`;
- для non-browser bearer client отсутствие `Origin` допустимо только при явном `authMethod=bearer` из проверенного auth context.

### Privacy

Первая версия публикует только агрегаты. В WebSocket payload не должны попадать:

- user ID;
- subject/issuer;
- username/display name;
- session ID;
- IP и User-Agent.

Такие данные можно использовать во внутренних logs с осторожностью, но не отправлять другим пользователям.

## WebSocket protocol v1

Все application messages — UTF-8 JSON. Добавить поле `protocolVersion: 1` в первый snapshot.

### Первоначальный snapshot

```json
{
  "type": "presence.snapshot",
  "protocolVersion": 1,
  "workspace": "default",
  "users": 2,
  "connections": 3,
  "sequence": 17,
  "occurredAt": "2026-08-05T12:00:00Z"
}
```

### Изменение агрегата

```json
{
  "type": "presence.changed",
  "workspace": "default",
  "users": 3,
  "connections": 4,
  "sequence": 18,
  "occurredAt": "2026-08-05T12:00:04Z"
}
```

Требования:

- новый connection сначала регистрируется, затем получает актуальный snapshot;
- после регистрации hub рассылает `presence.changed` всем connections workspace;
- закрытие одной из нескольких вкладок пользователя уменьшает только `connections`;
- `users` уменьшается только после закрытия последнего connection этого user;
- `sequence` монотонно увеличивается внутри процесса для каждого workspace;
- клиент должен считать snapshot/changed полной заменой агрегата, а не delta;
- неизвестные входящие text messages игнорируются или закрывают connection с protocol error по документированному правилу;
- binary application messages не поддерживаются.

Использовать WebSocket control frame ping/pong для heartbeat, а не собственные JSON-сообщения.

## Presence hub

Создать один singleton-компонент, зарегистрированный через Fx. Hub — единственный владелец mutable presence state.

Минимальная внутренняя модель:

```text
workspaceID
  -> userID
       -> connectionID -> connection state
```

`workspace identity` можно держать для ответа, но ключом изоляции должен быть UUID workspace.

### Требования к concurrency

- регистрация, удаление и snapshot потокобезопасны;
- broadcast одного медленного клиента не блокирует hub;
- у каждого connection есть bounded outgoing queue;
- при заполнении очереди slow consumer закрывается и удаляется;
- сетевое чтение/запись не выполняется под глобальным mutex;
- disconnect idempotent: повторная cleanup не меняет count ниже нуля;
- panic одной connection goroutine не должна оставлять ghost presence.

Предпочтителен event-loop hub с командами register/unregister/snapshot, либо аккуратная структура с mutex. Не создавать отдельный hub на каждый HTTP handler.

## Heartbeat и жизненный цикл

Добавить конфигурацию:

- `PRESENCE_ENABLED=false` по умолчанию до готовности deployment;
- `PRESENCE_MODE=single_instance`;
- `PRESENCE_PING_INTERVAL=25s`;
- `PRESENCE_PONG_TIMEOUT=60s`;
- `PRESENCE_WRITE_TIMEOUT=10s`;
- `PRESENCE_CONNECTION_QUEUE_SIZE=16`;
- `PRESENCE_MAX_CONNECTIONS_PER_USER=8`;
- `PRESENCE_MAX_CONNECTIONS_TOTAL=5000`.

Значения должны валидироваться на старте.

Алгоритм connection lifecycle:

1. HTTP auth.
2. Проверка Origin.
3. Авторизация workspace.
4. Проверка connection limits.
5. WebSocket upgrade.
6. Регистрация connection в hub.
7. Отправка snapshot.
8. Read loop обновляет read deadline при pong.
9. Write loop отправляет сообщения и ping.
10. При любой ошибке выполняется idempotent unregister и закрытие socket.

При остановке приложения Fx lifecycle должен:

- перестать принимать новые connections;
- отправить close frame с причиной service restart;
- закрыть все connections;
- дождаться ограниченное время завершения goroutines.

## Несколько вкладок и устройств

Presence агрегируется по `service_users.id`, а не по session ID. Поэтому:

- две вкладки одного пользователя: `users=1`, `connections=2`;
- два устройства одного пользователя: `users=1`, `connections=2`;
- два разных пользователя: `users=2`.

Connection ID генерируется backend и живёт только в памяти. Не принимать connection ID от клиента.

## Горизонтальное масштабирование

In-memory hub даёт правильный count только если все WebSocket connections попадают в один backend instance.

Первая версия обязана явно зафиксировать operational contract:

- presence включается только при одной реплике;
- ingress поддерживает WebSocket upgrade и достаточный idle timeout;
- health/readiness не объявляют feature распределённой;
- README предупреждает, что при нескольких репликах count будет локальным и feature нужно выключить.

Нельзя молча показывать локальный count как глобальный при нескольких репликах.

Распределённый presence — отдельная будущая задача. Не использовать PostgreSQL polling или Kafka только ради преждевременной поддержки масштаба.

## Наблюдаемость

Метрики с low-cardinality labels:

- `endge.presence.connections`;
- `endge.presence.users`;
- `endge.presence.connect_total{result=accepted|unauthorized|forbidden|limit|error}`;
- `endge.presence.disconnect_total{reason=client|timeout|slow_consumer|shutdown|error}`;
- `endge.presence.broadcast_total{result=success|dropped}`.

Workspace/user/connection IDs нельзя добавлять в metric labels.

Логи:

- INFO на старт/остановку hub и изменение feature mode;
- WARN для limit/slow consumer;
- ERROR для неожиданных ошибок upgrade/read/write;
- не логировать cookie, Authorization header или полный handshake.

## Структура backend-модулей

Рекомендуемое разделение:

- `internal/usecase/presence` — регистрация, агрегаты и application contract;
- `internal/usecase/ports/presence.go` — только если transport/hub разделяются портом;
- `internal/api/http/v1/presence` — WebSocket handler и protocol DTO;
- `internal/bootstrap` — singleton hub, lifecycle и handler binding;
- `internal/config` — presence settings.

WebSocket transport не должен обращаться к PostgreSQL напрямую. Авторизация выполняется через workspace use case.

## OpenAPI и документация

Обычный OpenAPI не полностью описывает WebSocket frames, поэтому:

- route и handshake errors описать в OpenAPI настолько, насколько позволяет генератор;
- JSON protocol и close codes описать в `README.md` или отдельной backend-документации;
- указать cookie auth, Origin, workspace RBAC и single-replica constraint;
- документировать необходимые ingress настройки без привязки к конкретному облаку.

Рекомендуемые close codes:

- `1000` — normal closure;
- `1001` — server shutdown;
- `1008` — policy violation;
- `1009` — message too large;
- `1011` — internal error.

## Тестовые сценарии

Тесты Go хранить по Go conventions рядом с package. Не создавать frontend fixtures.

Unit tests hub:

1. Первый connection создаёт `users=1`, `connections=1`.
2. Вторая вкладка того же user даёт `users=1`, `connections=2`.
3. Другой user даёт `users=2`.
4. Закрытие одной вкладки не уменьшает unique users.
5. Закрытие последней вкладки уменьшает unique users.
6. Повторный unregister idempotent.
7. Workspace UUID полностью изолируют counts.
8. Slow consumer не блокирует другие connections.
9. Лимиты connection работают без утечки регистрации.
10. Shutdown очищает состояние и завершает goroutines.

HTTP/WebSocket integration tests:

1. Нет auth — отказ до upgrade.
2. Нет доступа к workspace — `403` до upgrade.
3. Cookie auth с запрещённым Origin — `403`.
4. Cookie auth с разрешённым Origin — успешный upgrade.
5. Browser request без `X-Endge-Workspace` работает через path identity.
6. Snapshot соответствует зарегистрированным connections.
7. Ping/pong сохраняет connection.
8. Отсутствие pong удаляет connection после timeout.
9. Payload не содержит персональных данных.
10. `PRESENCE_ENABLED=false` возвращает документированный `404` или `503`; выбрать один вариант и использовать последовательно.

Race/leak проверки должны быть частью CI этой функции, даже если вручную они не запускаются при написании документа.

## Критерии приёмки

- Count считается внутри workspace, а не глобально.
- Workspace RBAC проверяется до upgrade.
- Cookie-auth WebSocket защищён проверкой Origin.
- Несколько вкладок одного пользователя не увеличивают `users`.
- Оборванное соединение исчезает после heartbeat timeout.
- Hub не блокируется медленным клиентом.
- Presence state не записывается в session table или PostgreSQL.
- Single-instance limitation явно отражён в config и документации.
- Протокол v1 имеет snapshot, changed, sequence и timestamp.
- В событиях нет персональных данных.
- Feature корректно выключается конфигурацией.

## Риски, которые нельзя скрывать

- Presence показывает открытое живое соединение, а не гарантированную активность человека за клавиатурой.
- Background browser tab может оставаться online. Idle/active status потребует отдельного протокола и продуктового определения.
- Presence не предотвращает одновременное редактирование одного документа.
- Несколько backend replicas без distributed hub дадут неверный глобальный count.
- Отсутствие явной Origin-проверки создаёт риск cross-site WebSocket hijacking при cookie auth.
