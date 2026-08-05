# Кеширование неизменяемых артефактов релизов

## Статус и приоритет

- Статус: требуется реализация.
- Приоритет: средний
- Область изменений: только backend.
- Миграция данных не требуется.

## Бизнес-контекст

Релиз — это зафиксированное состояние workspace на момент выбранного commit. После создания релиз не изменяется. Он используется для:

- скачивания переносимого JSON;
- восстановления workspace;
- последующей публикации или синхронизации между средами;
- запуска приложений из ранее опубликованного состояния.

Полный JSON релиза может быть большим. Повторное чтение одного и того же JSONB из PostgreSQL создаёт ненужную нагрузку на базу, аллокации и копирование байтов внутри процесса. Так как содержимое релиза неизменяемо, его можно лениво закешировать после первого чтения.

## Термины

- **Metadata релиза** — `id`, `workspaceId`, `identity`, название, описание, source commit, head sequence, schema version, checksum, автор и дата создания.
- **Artifact релиза** — поле `releases.data`, то есть сохранённый переносимый JSON.
- **Named release** — релиз с постоянным `identity`, например `production-2026-08-05`.
- **`last`** — вычисляемый alias, который каждый раз означает последний созданный релиз workspace. Сам alias неизменяемым не является.

## Текущее состояние

Основные точки кода:

- `internal/domain/entities/release.go` — модель релиза;
- `internal/repo/postgres/releases_repository.go` — чтение metadata и `data`;
- `internal/usecase/releases/queries.go` — workspace-scoped чтение;
- `internal/usecase/workspace_state/release_restore.go` — чтение artifact для plan/restore;
- `internal/api/http/v1/release/handler.go` — metadata, export и restore HTTP API;
- `internal/usecase/ports/releases.go` — текущий repository port;
- `migrations/000034_releases.sql` — неизменяемое хранилище релизов.

Сейчас `GetRelease` всегда использует `releaseSelect(true)` и загружает `releases.data`. Это происходит даже для `GET /api/v1/releases/{identity}`, хотя HTTP `Response` не содержит `data`. В результате metadata-запрос читает полный artifact без необходимости.

Endpoint export уже возвращает `ETag` из checksum, но не обрабатывает `If-None-Match`. Клиент поэтому не получает `304 Not Modified` и повторно скачивает те же байты.

## Цель

После реализации:

1. Metadata читается без `releases.data`.
2. Artifact читается отдельной операцией.
3. Первый запрос конкретного immutable artifact загружает его из PostgreSQL и сохраняет в ограниченный in-memory кеш.
4. Следующие запросы того же artifact обслуживаются из памяти процесса.
5. Параллельные cache miss одного ключа не создают несколько одинаковых запросов в PostgreSQL.
6. Кеш не может бесконтрольно занять всю память сервиса.
7. HTTP export поддерживает условные запросы по `ETag`.
8. Авторизация workspace выполняется до выдачи данных из кеша.

## Не входит в задачу

- кеширование live workspace или списков документов;
- кеширование списка релизов;
- Redis или другой распределённый кеш;
- изменение immutable-семантики релиза;
- изменение формата portable snapshot;
- фоновый прогрев всех релизов;
- удаление или TTL для данных в PostgreSQL.

## Обязательные архитектурные решения

### 1. Разделить metadata и artifact

Нельзя оставлять один метод, который всегда возвращает большое поле `data`.

Repository port должен различать как минимум:

- получение metadata конкретного релиза;
- разрешение `last` в конкретный релиз;
- чтение artifact конкретного релиза из постоянного хранилища;
- list metadata без artifact.

Допустимая форма application-модели:

```go
type ReleaseArtifact struct {
    ReleaseID   string
    WorkspaceID string
    Identity    string
    Checksum    string
    Data        json.RawMessage
}
```

Точные имена методов можно выбрать по стилю проекта, но семантика должна быть явной. Не использовать булевый `includeData` за пределами PostgreSQL implementation: application layer не должен случайно запросить тяжёлое поле.

### 2. Один владелец чтения artifact

Export, restore plan и restore должны использовать один application-сервис чтения release artifact. Нельзя создавать отдельный кеш в HTTP handler и второй кеш в `workspace_state`.

Рекомендуемая схема:

```text
HTTP / release restore
        |
        v
ReleaseArtifactReader
        |
        +-- bounded in-memory cache
        |
        +-- PostgreSQL artifact repository
```

`ReleaseArtifactReader` получает уже разрешённые `workspaceID` и release metadata. Проверка доступа остаётся в use case до чтения кеша.

### 3. Ключ кеша

Использовать фактический immutable идентификатор, а не только строковый alias:

```text
workspace UUID + release UUID + checksum
```

`last` нельзя хранить как самостоятельный cache key. Для каждого запроса `last` нужно:

1. определить актуальный последний релиз в PostgreSQL;
2. получить его UUID и checksum;
3. прочитать artifact по обычному immutable ключу.

Named release может разрешаться напрямую, так как `(workspace_id, identity)` уникален и после создания не меняется.

### 4. Ограничение памяти

Кеш должен быть bounded LRU с учётом размера `Data` в байтах.

Добавить конфигурацию с валидацией и документированными defaults:

- `RELEASE_ARTIFACT_CACHE_ENABLED=true`;
- `RELEASE_ARTIFACT_CACHE_MAX_BYTES=67108864` — 64 MiB;
- `RELEASE_ARTIFACT_CACHE_MAX_ITEM_BYTES=16777216` — 16 MiB.

Правила:

- значение `MAX_BYTES <= 0` при включённом кеше является ошибкой конфигурации;
- artifact больше `MAX_ITEM_BYTES` корректно возвращается клиенту, но не сохраняется в памяти;
- вставка нового элемента вытесняет самые давно не использованные элементы до соблюдения `MAX_BYTES`;
- размер считается по фактической длине byte slice, без приблизительного количества объектов;
- операции потокобезопасны;
- вызывающий код не должен иметь возможности изменить bytes, уже находящиеся в кеше. Нужна неизменяемость владения или защитное копирование на границе.

Не использовать TTL как главный механизм. Named release не устаревает; ограничением должны быть память и LRU.

### 5. Защита от cache stampede

Если несколько запросов одновременно читают отсутствующий artifact, PostgreSQL должен получить один запрос, а остальные запросы должны дождаться того же результата.

Требования:

- дедупликация выполняется по полному cache key;
- ошибка загрузки не кешируется как успешное значение;
- отмена одного HTTP request не должна повреждать ожидание других запросов;
- после завершения загрузки временное состояние ожидания удаляется.

Можно использовать небольшой keyed-flight механизм или подходящий пакет, если зависимость оправдана. Нельзя держать глобальную блокировку во время SQL-запроса.

### 6. Поведение при создании релиза

Кеш должен быть lazy. Создание релиза сохраняет artifact в PostgreSQL, но не обязано сразу занимать память кеша.

После успешного INSERT вернуть metadata без повторного чтения полного `data`. Если repository для корректного ответа делает SELECT, SELECT должен быть metadata-only.

### 7. HTTP cache contract

Для `GET /api/v1/releases/{identity}/export`:

- после успешной аутентификации и проверки workspace вернуть `ETag: "<checksum>"`;
- распознавать `If-None-Match`, включая стандартный список значений и weak ETag;
- при совпадении вернуть `304` без body;
- для named release вернуть `Cache-Control: private, no-cache`, чтобы browser каждый раз перепроверял auth/RBAC и затем мог получить дешёвый `304`;
- добавить `Vary: X-Endge-Workspace, Authorization, Cookie`;
- для `identity=last` также вернуть `Cache-Control: private, no-cache`; его ETag меняется после появления нового релиза;
- режим `download=true` не должен менять checksum или cache key.

Проверка `If-None-Match` не должна обходить auth/RBAC. Нельзя возвращать `304` неавторизованному пользователю только потому, что он угадал checksum.

Не использовать длительный browser `max-age` даже для immutable named release: право пользователя читать workspace может быть отозвано после первого ответа. В этой задаче ускорение обеспечивает server-side artifact cache, а client cache обязан revalidate доступ.

### 8. Ошибки и целостность

- `404` для отсутствующего релиза не кешируется навсегда;
- PostgreSQL остаётся источником истины;
- при загрузке artifact проверить, что обязательные поля metadata присутствуют;
- checksum, сохранённый при создании релиза, является identity содержимого;
- если принято решение пересчитывать checksum на чтении, делать это только на cache miss и считать несовпадение внутренней ошибкой/нарушением целостности. Не пересчитывать SHA-256 на каждом cache hit.

## Метрики

Добавить OpenTelemetry-метрики без workspace ID, release identity или UUID в labels, чтобы не создавать высокую cardinality:

- `endge.release_artifact_cache.requests_total{result=hit|miss|bypass|error}`;
- `endge.release_artifact_cache.evictions_total`;
- `endge.release_artifact_cache.items`;
- `endge.release_artifact_cache.bytes`;
- `endge.release_artifact_cache.load_duration_ms`.

Допустимые low-cardinality attributes: `result`, `operation=export|restore_plan|restore`.

Не логировать содержимое artifact. В debug/error логах допустимы release UUID, workspace UUID, checksum и размер.

## Изменения конфигурации и документации

Нужно обновить:

- `internal/config/config.go`;
- `.env.development.example`;
- `.env.production.example`;
- `README.md` с назначением параметров и поведением при нескольких репликах;
- OpenAPI для `If-None-Match`, `304`, `Cache-Control` и `ETag`.

При нескольких репликах кеш прогревается независимо в каждой реплике. Это корректное поведение и должно быть описано. Не добавлять Redis без отдельного требования.

## Ожидаемая последовательность реализации

1. Разделить SQL и repository port для metadata/artifact.
2. Перевести существующие metadata endpoints на metadata-only чтение.
3. Создать единый `ReleaseArtifactReader`.
4. Реализовать bounded LRU и keyed-flight.
5. Перевести export и release restore на reader.
6. Добавить HTTP conditional GET.
7. Добавить конфигурацию, метрики и документацию.
8. Добавить focused unit/integration tests.

## Проверки и тестовые сценарии

Go tests размещать по обычным Go conventions рядом с package.

Обязательные сценарии:

1. Metadata GET не выбирает и не возвращает `data`.
2. Первый artifact read вызывает repository один раз и возвращает bytes.
3. Повторный read того же ключа не вызывает repository.
4. Десять параллельных read одного ключа создают одну загрузку.
5. Одинаковый release identity в разных workspace не пересекается в кеше.
6. Изменение результата, полученного вызывающим кодом, не портит кеш.
7. LRU вытесняет старые элементы по суммарному размеру.
8. Слишком большой artifact обслуживается с `bypass` и не кешируется.
9. Ошибка repository не сохраняется как cache hit.
10. `last` после создания нового релиза указывает на новый release UUID.
11. Named release с совпадающим `If-None-Match` возвращает `304` после auth.
12. Неавторизованный запрос с правильным ETag получает `401/403`, а не `304`.
13. Restore plan и restore читают тот же кеш, что export.
14. Выключенный кеш оставляет функциональное поведение без изменений.

## Критерии приёмки

- В production path нет metadata-запроса, который без необходимости читает `releases.data`.
- Artifact кешируется только после успешного первого чтения.
- Память кеша ограничена конфигурацией и фактически соблюдает лимит.
- `last` не становится stale alias.
- Export и restore используют одного владельца кеша.
- RBAC выполняется до доступа к cached bytes.
- Conditional GET работает и описан в OpenAPI.
- Метрики не содержат high-cardinality labels.
- Существующая семантика create/list/get/export/restore релизов не сломана.

## Риски, которые нельзя скрывать

- In-memory кеш ускоряет только повторные обращения к той же реплике.
- Большой лимит кеша увеличивает RSS Go-процесса; лимит должен соответствовать container memory limit.
- Кеширование полного `entities.Release` вместо artifact сохранит лишние данные и снова смешает metadata с тяжёлым payload.
- Кеширование alias `last` приведёт к выдаче старого релиза.
- Длительный browser `max-age` опасен для workspace-scoped authenticated response, потому что может обойти повторную проверку отозванного RBAC.
