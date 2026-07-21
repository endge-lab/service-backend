# 04. Workspaces Foundation

## Цель

Реализовать `RWorkspace` как корневой организационный scope домена: migration, entity, repository, usecase и HTTP API.

Workspace хранит полную исходную `EndgeConfiguration`. Это единственный из четырёх configuration layers, который не является patch. Project, Environment и Tenant хранят собственные `EndgeConfigurationContribution` и уточняют либо полностью заменяют результат предыдущего слоя.

На текущем этапе backend открытый: authentication, users, memberships и роли не реализовывать. Все клиенты имеют полный доступ ко всем workspace.

## Таблица `workspaces`

```sql
CREATE TABLE workspaces (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  identity TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  configuration JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (btrim(identity) <> ''),
  CHECK (btrim(display_name) <> ''),
  CHECK (jsonb_typeof(configuration) = 'object')
);
```

Не раскладывать `vars`, locales, themes, SSE и adapter settings по отдельным колонкам. Persisted contract и frontend contract должны иметь одну nested-структуру `configuration`.

## Полная `EndgeConfiguration`

```json
{
  "vars": [],
  "locales": [
    {
      "code": "ru",
      "displayName": "Русский",
      "shortLabel": "RU",
      "direction": "ltr"
    },
    {
      "code": "en",
      "displayName": "English",
      "shortLabel": "EN",
      "direction": "ltr"
    }
  ],
  "defaultLocale": "ru",
  "fallbackLocale": "ru",
  "themes": [
    { "identity": "light", "displayName": "Светлая" },
    { "identity": "dark", "displayName": "Тёмная" }
  ],
  "defaultTheme": "light",
  "defaultAuthProfileIdentity": null,
  "sfcAdapterIds": ["native-vue"],
  "defaultSfcAdapterId": "native-vue"
}
```

Это одновременно default для новой записи. Поле `sse` optional и при отсутствии настройки не включается в JSON:

```json
{
  "sse": {
    "url": "{ENDPOINT_SSE}",
    "authMode": "inherit",
    "authProfileIdentity": null,
    "manualToken": null
  }
}
```

Допустимые значения:

```text
locales[].direction: ltr | rtl
sse.authMode: inherit | profile | manual | none
```

Usecase обязан проверять:

- `vars`, `locales`, `themes`, `sfcAdapterIds` являются arrays;
- `locales`, `themes` и `sfcAdapterIds` не пустые;
- `vars[].name`, `locales[].code`, `themes[].identity` и элементы `sfcAdapterIds` непустые и уникальные внутри массива;
- `defaultLocale` и `fallbackLocale` входят в `locales[].code`;
- `defaultTheme` входит в `themes[].identity`;
- `defaultSfcAdapterId` входит в `sfcAdapterIds`;
- `defaultAuthProfileIdentity` и `sse.authProfileIdentity` имеют значение `string | null`;
- при `sse.authMode=profile` указан `sse.authProfileIdentity`;
- при `sse.authMode=manual` значение `manualToken` обрабатывается как secret.

После задачи `18-auth-profiles-foundation` auth profile identities валидируются внутри текущего workspace. В portable configuration сохраняется стабильный identity, а не UUID relation из конкретной БД.

`sse.manualToken` нельзя логировать, добавлять в traces/errors или включать в portable/debug dump. Если backend поддерживает encryption at rest, сохранять secret через этот механизм.

## Configuration cascade

Backend должен использовать тот же порядок, что и frontend Core:

```text
Workspace.configuration
  -> Project.configuration
  -> Environment.configuration
  -> Tenant.configuration
  = effective EndgeConfiguration
```

Эффективная конфигурация зависит от полного execution context и вычисляется при boot/build. Её нельзя сохранять в отдельную таблицу или обратно в любую из четырёх сущностей.

Workspace, Project, Environment и Tenant не образуют жёсткую parent chain через foreign keys друг на друга. Project и Environment могут использоваться в разных execution contexts. Все три дочерних слоя принадлежат Workspace, а конкретное сочетание выбирается при запуске.

## Связь с доменом

- добавить `workspace_id UUID NOT NULL REFERENCES workspaces(id)` в `projects`, `environments`, `tenants`, `folders` и остальные workspace-scoped таблицы;
- uniqueness стабильных identities задавать внутри workspace, если отдельная задача не требует глобальной уникальности;
- нельзя доверять `workspaceId` или `workspaceIdentity` из body доменных документов: scope приходит из request context;
- не добавлять `tenant_id`, `project_id` или `environment_id` в сущность только ради configuration resolution;
- правила внешних ключей, dependency validation и portable import реализуются по задаче `07-domain-relations-and-portable-import`.

## Entity и usecase

Создать `RWorkspace` и port:

```text
WorkspacesRepository
```

Минимальные operations:

```text
Create(input) -> workspace
List() -> workspaces
GetByIdentity(identity) -> workspace
Update(identity, patch) -> workspace
```

Update является partial update верхнего уровня. `identity`, `id` и `createdAt` неизменяемы. Если передано поле `configuration`, оно заменяет полную root configuration и проходит полную validation. Частичный JSON merge внутри `configuration` не выполнять.

Удаление workspace в этой задаче не реализовывать: сначала должна быть определена cascade/archive policy для всего домена.

## HTTP API

Эти endpoints не требуют `X-Endge-Workspace`, потому что они выбирают или создают сам workspace.

```text
GET   /api/v1/workspaces
POST  /api/v1/workspaces
GET   /api/v1/workspaces/:workspace_identity
PATCH /api/v1/workspaces/:workspace_identity
```

Create request:

```json
{
  "identity": "default",
  "displayName": "Default workspace",
  "configuration": {
    "vars": [],
    "locales": [
      { "code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr" },
      { "code": "en", "displayName": "English", "shortLabel": "EN", "direction": "ltr" }
    ],
    "defaultLocale": "ru",
    "fallbackLocale": "ru",
    "themes": [
      { "identity": "light", "displayName": "Светлая" },
      { "identity": "dark", "displayName": "Тёмная" }
    ],
    "defaultTheme": "light",
    "defaultAuthProfileIdentity": null,
    "sfcAdapterIds": ["native-vue"],
    "defaultSfcAdapterId": "native-vue"
  }
}
```

Поле `configuration` можно не передавать при create: backend обязан поставить system default выше.

Response:

```json
{
  "id": "00000000-0000-4000-8000-000000000001",
  "identity": "default",
  "displayName": "Default workspace",
  "configuration": {
    "vars": [],
    "locales": [
      { "code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr" },
      { "code": "en", "displayName": "English", "shortLabel": "EN", "direction": "ltr" }
    ],
    "defaultLocale": "ru",
    "fallbackLocale": "ru",
    "themes": [
      { "identity": "light", "displayName": "Светлая" },
      { "identity": "dark", "displayName": "Тёмная" }
    ],
    "defaultTheme": "light",
    "defaultAuthProfileIdentity": null,
    "sfcAdapterIds": ["native-vue"],
    "defaultSfcAdapterId": "native-vue"
  },
  "createdAt": "2026-07-16T10:00:00Z",
  "updatedAt": "2026-07-16T10:00:00Z"
}
```

`PATCH` принимает `displayName` и/или полную `configuration`. `sse.manualToken` в обычных responses должен быть redacted согласно общей secret policy.

## Tests

Минимально проверить:

- create без `configuration` использует `ru/en`, `light/dark` и `light` по умолчанию;
- nested configuration проходит repository round-trip без раскладывания по legacy columns;
- locale, theme и adapter invariants;
- optional SSE и secret redaction;
- partial workspace update не выполняет скрытый merge внутри configuration;
- duplicate workspace identity;
- list/get/create/update HTTP scenarios.

## Acceptance Criteria

- migration создаёт одну nested `configuration JSONB`, без legacy columns `vars`, `locales`, `themes` и подобных;
- default configuration содержит locales `ru/en`, themes `light/dark`, default locale `ru` и default theme `light`;
- list/get возвращают все workspaces без проверки пользователя;
- configuration invariants валидируются в domain/usecase layer;
- project, environment и tenant остаются отдельными workspace-scoped layers;
- есть repository/usecase/HTTP tests и проходит `go test ./...`.
