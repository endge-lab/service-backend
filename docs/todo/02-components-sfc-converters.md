# 02. Components SFC + Converters

## Исходные условия

Считать, что в сервисе есть только базовый Go backend template и результат задачи `01-projects-folders-foundation`.

Не опираться на старые миграции, старые таблицы или старую модель `components/converters`, если она уже есть в репозитории. Целевая модель описана в этом документе.

## Главное правило идентификаторов

Все UUID `id` генерируются на стороне backend.

Клиент явно передает только человекочитаемые `identity`:

- `project_identity` в path;
- `folderIdentity` в request body;
- `component identity`;
- `converter identity`.

Внутри БД связи хранятся через UUID foreign keys, но HTTP API должен резолвить их по identity. Клиент не должен вручную передавать `projectId` или `folderId`.

## Контекст

Component - это конфигурационный артефакт, который описывает UI-компонент в проекте. На первом этапе нужно поддержать только один тип компонента:

```text
component-sfc
```

Component SFC - это persisted authoring source. Сервис должен хранить исходник компонента и метаданные, но не должен компилировать, исполнять или валидировать Vue/SFC runtime.

Converter - это именованный конфигурационный артефакт, который описывает переиспользуемое преобразование данных. На этом этапе converter является catalog/config record: сервис хранит его source/config, но не исполняет код преобразования.

## Модель данных

Нужно создать новые миграции для таблиц `components` и `converters`.

### components

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE RESTRICT,
identity TEXT NOT NULL,
display_name TEXT NOT NULL,
description TEXT NULL,
component_type TEXT NOT NULL,
source TEXT NOT NULL,
source_format TEXT NOT NULL DEFAULT 'sfc',
props_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
bindings JSONB NOT NULL DEFAULT '{}'::jsonb,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
active BOOLEAN NOT NULL DEFAULT TRUE,
deleted_at TIMESTAMPTZ NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Ограничения:

- `component_type` на этом этапе допускает только `component-sfc`.
- `source_format` на этом этапе допускает только `sfc`.
- `identity` уникален внутри проекта.
- `identity` не пустой после trim.
- `display_name` не пустой после trim.
- `source` не пустой после trim.
- `folder_id` должен ссылаться на папку того же `project_id`.
- `folder_id` должен ссылаться на папку с `entity_type = components`.
- `created_at` нельзя менять через API.
- `updated_at` обновляется при каждом update/soft-delete/restore.

Важно: не использовать поля с названием `jsx_script` или старые payload-specific поля. Source компонента должен храниться в явном поле `source`.

### converters

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE RESTRICT,
identity TEXT NOT NULL,
display_name TEXT NOT NULL,
description TEXT NULL,
converter_type TEXT NOT NULL,
source JSONB NOT NULL DEFAULT '{}'::jsonb,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
active BOOLEAN NOT NULL DEFAULT TRUE,
deleted_at TIMESTAMPTZ NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Ограничения:

- `identity` уникален внутри проекта.
- `identity` не пустой после trim.
- `display_name` не пустой после trim.
- `converter_type` не пустой после trim.
- `folder_id` должен ссылаться на папку того же `project_id`.
- `folder_id` должен ссылаться на папку с `entity_type = converters`.
- System converter нельзя hard-delete.

На этом этапе `source` у converter - это JSON config/source descriptor. Сервис не должен выполнять этот source.

## API

Все endpoints должны быть под `/api/v1`.

### Components

```text
GET    /api/v1/projects/:project_identity/components?folder_identity=...&component_type=component-sfc
POST   /api/v1/projects/:project_identity/components
GET    /api/v1/projects/:project_identity/components/:component_identity
PATCH  /api/v1/projects/:project_identity/components/:component_identity
DELETE /api/v1/projects/:project_identity/components/:component_identity
POST   /api/v1/projects/:project_identity/components/:component_identity/restore
DELETE /api/v1/projects/:project_identity/components/:component_identity/hard
```

Create request:

```json
{
  "folderIdentity": "root-components",
  "identity": "user-card",
  "displayName": "User Card",
  "description": "SFC component for user card",
  "componentType": "component-sfc",
  "source": "<template><div>{{ props.name }}</div></template>",
  "propsSchema": {
    "name": {
      "type": "string",
      "required": true
    }
  },
  "bindings": {},
  "meta": {},
  "active": true
}
```

Response:

```json
{
  "id": "00000000-0000-4000-8000-000000000101",
  "projectIdentity": "demo-project",
  "folderIdentity": "root-components",
  "identity": "user-card",
  "displayName": "User Card",
  "description": "SFC component for user card",
  "componentType": "component-sfc",
  "source": "<template><div>{{ props.name }}</div></template>",
  "sourceFormat": "sfc",
  "propsSchema": {},
  "bindings": {},
  "meta": {},
  "active": true,
  "deletedAt": null,
  "createdAt": "2026-07-08T10:00:00Z",
  "updatedAt": "2026-07-08T10:00:00Z"
}
```

### Converters

```text
GET    /api/v1/projects/:project_identity/converters?folder_identity=...
POST   /api/v1/projects/:project_identity/converters
GET    /api/v1/projects/:project_identity/converters/:converter_identity
PATCH  /api/v1/projects/:project_identity/converters/:converter_identity
DELETE /api/v1/projects/:project_identity/converters/:converter_identity
POST   /api/v1/projects/:project_identity/converters/:converter_identity/restore
DELETE /api/v1/projects/:project_identity/converters/:converter_identity/hard
```

Create request:

```json
{
  "folderIdentity": "root-converters",
  "identity": "date-to-display",
  "displayName": "Date to Display",
  "description": "Formats date values for UI output",
  "converterType": "format",
  "source": {
    "kind": "template",
    "template": "{{ value }}"
  },
  "isSystem": false,
  "meta": {},
  "active": true
}
```

## Поведение

- Обычные list/get endpoints не возвращают soft-deleted записи.
- `DELETE /api/v1/projects/:project_identity/components/:component_identity` делает soft-delete.
- `DELETE /api/v1/projects/:project_identity/converters/:converter_identity` делает soft-delete.
- Hard-delete физически удаляет запись.
- Hard-delete system converter запрещен.
- Update должен заменять весь editable payload сущности, кроме `id`, `identity`, `created_at`, `deleted_at`.
- Если указан `folderIdentity` не того проекта или не того `entity_type`, вернуть validation error.

## Ошибки

Использовать единый JSON-формат:

```json
{
  "code": "validation_error",
  "message": "Validation error",
  "details": {}
}
```

Минимальные коды:

```text
validation_error
not_found
identity_conflict
folder_entity_type_mismatch
system_converter_delete_forbidden
internal_error
```

## Слои

Реализовать задачу через текущую clean architecture структуру:

- domain entities: `Component`, `Converter`;
- usecase ports: `ComponentsRepository`, `ConvertersRepository`;
- postgres repositories;
- HTTP handlers and transport DTO;
- bootstrap registration through `fx`;
- sqlc queries;
- migrations.

Usecase слой не должен импортировать postgres package.

## Tests

Минимально нужны:

- validation tests для create/update component;
- validation tests для create/update converter;
- tests на `identity` conflict внутри одного проекта;
- tests на проверку folder `entity_type`;
- HTTP handler tests для create/list/get/update/delete/restore;
- `go test ./...` должен проходить.

## Acceptance Criteria

- Новые миграции создают целевую модель из этого документа.
- Можно создать `component-sfc` с source в поле `source`.
- Сервис не компилирует и не исполняет SFC source.
- Можно создать converter с JSON `source`.
- Сервис не исполняет converter source.
- Клиент не передает UUID при создании component/converter.
- Связи в публичном API задаются через identity.
- Нельзя создать component/converter в папке чужого проекта.
- Нельзя создать component в folder с `entity_type != components`.
- Нельзя создать converter в folder с `entity_type != converters`.
- Soft-deleted записи не попадают в обычные списки.
- `go test ./...` проходит.
