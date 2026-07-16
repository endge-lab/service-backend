# 04. Workspaces Foundation

## Цель

Реализовать `RWorkspace` как корневой scope домена: migration, entity, repository, usecase и HTTP API.

На текущем этапе backend открытый: authentication, users, memberships и роли не реализовывать. Все клиенты имеют полный доступ ко всем workspace.

## Таблица `workspaces`

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL,
display_name TEXT NOT NULL,
vars JSONB NOT NULL DEFAULT '[]'::jsonb,
sse JSONB NULL,
sse_auth_profile_id UUID NULL,
locales JSONB NOT NULL,
default_locale TEXT NOT NULL,
fallback_locale TEXT NOT NULL,
default_auth_profile_id UUID NULL,
sfc_adapter_ids JSONB NOT NULL DEFAULT '["native-vue"]'::jsonb,
default_sfc_adapter_id TEXT NOT NULL DEFAULT 'native-vue',
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Ограничения:

- `identity` уникален и не пуст после trim;
- `display_name` не пуст после trim;
- `vars`, `locales`, `sfc_adapter_ids` — JSON arrays;
- `locales` и `sfc_adapter_ids` не могут быть пустыми;
- `default_locale` и `fallback_locale` должны присутствовать в `locales[].code`;
- `default_sfc_adapter_id` должен присутствовать в `sfc_adapter_ids`.

Точные JSON-форматы:

```json
{
  "vars": [{ "name": "ENDPOINT_API", "defaultValue": "https://example.test" }],
  "sse": {
    "url": "{ENDPOINT_SSE}",
    "authMode": "inherit",
    "authProfileIdentity": null,
    "manualToken": null
  },
  "locales": [
    { "code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr" }
  ],
  "sfcAdapterIds": ["native-vue"]
}
```

Допустимые значения:

```text
locales[].direction: ltr | rtl
sse.authMode: inherit | profile | manual | none
```

`vars[].name`, `locales[].code` и элементы `sfc_adapter_ids` должны быть непустыми и уникальными внутри своих массивов. `sse` целиком может быть `NULL`.

HTTP API принимает `defaultAuthProfileIdentity` и `sse.authProfileIdentity`, но usecase разрешает их по `(workspace_id, identity)` и сохраняет UUID в `default_auth_profile_id` и `sse_auth_profile_id`. В `sse JSONB` поле `authProfileIdentity` не дублировать. До выполнения задачи `18-auth-profiles-foundation` оба поля должны быть `NULL`; внешние ключи и conditional constraints добавляются migration из задачи №18 после создания `auth_profiles`.

Для `sse.authMode=profile` обязателен `sse_auth_profile_id`; при других режимах он должен быть `NULL`. `default_auth_profile_id` может быть `NULL`. Удаление используемого auth profile должно блокироваться через `ON DELETE RESTRICT` и domain error `auth_profile_in_use`.

`sse.manualToken` является sensitive value: не логировать, не добавлять в traces/errors и не включать в debug dump. Если в проекте уже есть механизм шифрования secrets, хранить значение через него.

## Связь с доменом

- добавить `workspace_id UUID NOT NULL REFERENCES workspaces(id)` в `projects`;
- уникальность проекта изменить на `(workspace_id, identity)`;
- project usecase/repository всегда получает `workspaceID` и фильтрует по нему;
- будущие workspace-scoped таблицы должны иметь `workspace_id` напрямую, если `project_id` у них optional;
- нельзя доверять `workspaceId/workspaceIdentity` из body доменных документов: scope приходит из request context.
- правила внешних ключей, dependency validation и portable import реализуются по задаче `07-domain-relations-and-portable-import`.

## Entity и usecase

Создать `RWorkspace` с полями, перечисленными выше, и ports:

```text
WorkspacesRepository
```

Минимальные usecase operations:

```text
Create(input) -> workspace
List() -> workspaces
GetByIdentity(identity) -> workspace
Update(identity, patch) -> workspace
```

Update является partial update; `identity`, `id` и `createdAt` неизменяемы. Удаление workspace в этой задаче не реализовывать: сначала должна быть определена cascade/archive policy для всего домена.

## HTTP API

Эти endpoints не требуют `X-Endge-Workspace`, потому что они выбирают или создают сам workspace. API принимает и возвращает camelCase поля.

### `GET /api/v1/workspaces`

Возвращает список всех workspace. Query и body parameters отсутствуют.

Response `200 OK`:

```json
{
  "items": [
    {
      "id": "00000000-0000-4000-8000-000000000001",
      "identity": "default",
      "displayName": "Default workspace",
      "vars": [],
      "sse": null,
      "locales": [
        { "code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr" }
      ],
      "defaultLocale": "ru",
      "fallbackLocale": "ru",
      "defaultAuthProfileIdentity": null,
      "sfcAdapterIds": ["native-vue"],
      "defaultSfcAdapterId": "native-vue",
      "createdAt": "2026-07-16T10:00:00Z",
      "updatedAt": "2026-07-16T10:00:00Z"
    }
  ]
}
```

### `POST /api/v1/workspaces`

Создаёт новый workspace. `identity` должен быть уникальным.

Request body:

```json
{
  "identity": "default",
  "displayName": "Default workspace",
  "vars": [
    { "name": "ENDPOINT_API", "defaultValue": "https://example.test" }
  ],
  "sse": {
    "url": "{ENDPOINT_SSE}",
    "authMode": "inherit",
    "authProfileIdentity": null,
    "manualToken": null
  },
  "locales": [
    { "code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr" }
  ],
  "defaultLocale": "ru",
  "fallbackLocale": "ru",
  "defaultAuthProfileIdentity": null,
  "sfcAdapterIds": ["native-vue"],
  "defaultSfcAdapterId": "native-vue"
}
```

Response `201 Created`: созданный workspace, включая `id`, `createdAt` и `updatedAt`. Sensitive поле `sse.manualToken` в response не возвращать. Если `identity` уже занят — `409 Conflict`.

При создании workspace ещё не существует auth profile в его scope, поэтому `defaultAuthProfileIdentity` и `sse.authProfileIdentity` в POST должны быть `null`. После создания профиля их можно назначить через PATCH; usecase разрешает identity в UUID transactionally.

### `GET /api/v1/workspaces/:workspace_identity`

Возвращает один workspace по его стабильному `identity`.

Path parameter:

```text
workspace_identity = default
```

Пример запроса:

```http
GET /api/v1/workspaces/default
```

Response `200 OK`: workspace в том же формате, что элемент `items` в list response. Если запись не найдена — `404 Not Found`.

### `PATCH /api/v1/workspaces/:workspace_identity`

Частично обновляет существующий workspace. В body передаются только изменяемые поля; `id`, `identity`, `createdAt` и `updatedAt` клиент не передаёт.

Path parameter:

```text
workspace_identity = default
```

Request body example:

```json
{
  "displayName": "Main workspace",
  "fallbackLocale": "en",
  "locales": [
    { "code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr" },
    { "code": "en", "displayName": "English", "shortLabel": "EN", "direction": "ltr" }
  ]
}
```

Переданные массивы `vars`, `locales` и `sfcAdapterIds` заменяются целиком; значение `sse: null` удаляет SSE-конфигурацию.

Response `200 OK`: обновлённый workspace без `sse.manualToken`. Если запись не найдена — `404 Not Found`; если нарушены locale/SFC invariants — `400 Bad Request`.

Поле `id` возвращается как UUID string; в URLs и workspace header используется стабильный `identity`. Не преобразовывать UUID в number и не генерировать отдельный numeric ID в этой задаче.

## Acceptance Criteria

- migration `000002_init_workspaces.sql` больше не содержит TODO;
- таблицы пользователей, memberships и ролей не создаются;
- list/get возвращают все workspaces без проверки пользователя;
- locale и SFC adapter invariants валидируются в usecase;
- auth profile identities из HTTP не сохраняются как relation strings: после задачи №18 они разрешаются в UUID foreign keys;
- project identity изолирован внутри workspace;
- есть repository/usecase/HTTP tests и проходит `go test ./...`.
