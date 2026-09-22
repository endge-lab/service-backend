# Техническая постановка: Endge Configuration Backend на Go

## Контекст

Нужно реализовать самостоятельный Go-сервис, который заменяет текущий storage-слой конфигуратора Endge. У разработчика нет доступа к исходным репозиториям Payload, frontend-конфигуратора и Pilot, поэтому этот документ должен считаться самодостаточным ТЗ первого этапа.

Сервис должен быть монолитным backend API на Go + PostgreSQL. Авторизация, телеметрия, Redpanda/Kafka и отдельный админ-интерфейс на первом этапе не требуются.

Основные задачи:

- хранить конфигурационные сущности Endge-domain;
- отдавать полный снимок домена для frontend-конфигуратора;
- поддерживать CRUD по конфигурационным сущностям;
- поддерживать папки, мягкое удаление, восстановление, жесткое удаление и версии;
- сохранить бизнес-логику текущего конфигурационного storage, но не копировать физическую схему Payload CMS один-в-один.

## Термины

- `id` - внутренний UUID документа в новом Go-сервисе.
- `identity` - стабильный человекочитаемый уникальный идентификатор документа внутри коллекции. Например `root-components`, `default`, `UserTable`.
- `collection` - имя коллекции в API, например `components`, `queries`, `folders`.
- `folder_id` - ссылка на папку, в которой лежит документ.
- `project_id` - ссылка на проект, если документ относится к проекту.
- `deleted_at` - признак мягкого удаления. Если `deleted_at IS NOT NULL`, документ считается удаленным.
- `soft-deleted` - системная папка-корзина.
- `meta` - произвольные метаданные в `jsonb`.
- `schema` - произвольное тело доменной сущности в `jsonb`.

## Общие требования

1. Все публичные API должны быть под префиксом `/api`.
2. Все ответы должны быть JSON.
3. Все ошибки должны возвращаться в едином формате:

```json
{
  "code": "not_found",
  "message": "Document not found",
  "details": {}
}
```

4. Для всех сущностей, где есть `identity`, значение `identity` должно быть уникальным внутри своей коллекции.
5. Нельзя создать документ без обязательных полей.
6. Нельзя создать два документа с одинаковым `identity` в одной коллекции.
7. Нельзя обновить, удалить, восстановить или переместить несуществующий документ.
8. Мягко удаленный документ не должен попадать в обычные списки по умолчанию.
9. Жесткое удаление должно физически удалять запись из БД.
10. Все операции записи должны обновлять `updated_at`.
11. Нельзя менять `created_at` через API.
12. Операции import/restore версии должны выполняться в транзакции.

## Технический стек

- Go
- PostgreSQL 16+
- Миграции через goose.
- UUID как внутренние id.
- Сложные вложенные структуры хранить в `jsonb`.
- HTTP framework: Fiber или стандартный `net/http`, но REST-контракт должен быть сохранен.

## Таблицы

### Общие поля

Большинство таблиц должны иметь общий набор полей:

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
active BOOLEAN NOT NULL DEFAULT TRUE,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
author TEXT NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Не все таблицы обязаны иметь все поля. Ниже описаны отличия.

### folders

Дерево папок для всех типов сущностей.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
entity_type TEXT NOT NULL,
parent_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
is_root BOOLEAN NOT NULL DEFAULT FALSE,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Допустимые `entity_type`:

```text
projects
components
queries
scenarios
actions
types
converters
parameters
filters
views
integrations
environments
tenants
behavior-bindings
presentation-bindings
policies
styles
page-templates
pages
navigations
settings
vocabs
i18n-bundles
system
```

Системные папки, которые нужно создать seed-миграцией:

```text
root-projects
root-components
root-queries
root-scenarios
root-actions
root-types
root-converters
root-parameters
root-filters
root-views
root-integrations
root-environments
root-tenants
root-bindings
root-presentation-bindings
root-policies
root-styles
root-page-templates
root-pages
root-navigations
root-settings
root-vocabs
root-i18n-bundles
soft-deleted
```

Правила:

- системную папку нельзя жестко удалить;
- root-папку нельзя переместить;
- папку нельзя сделать дочерней самой себе;
- папку нельзя сделать дочерней своему потомку;
- при мягком удалении папка переносится под `soft-deleted` или получает `deleted_at`.

### projects

Проекты/контексты конфигуратора.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
extend_settings BOOLEAN NOT NULL DEFAULT FALSE,
settings_id UUID NULL REFERENCES settings(id) ON DELETE SET NULL,
navigation_id UUID NULL REFERENCES navigations(id) ON DELETE SET NULL,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
allowed_environment_ids UUID[] NOT NULL DEFAULT '{}',
deleted_at TIMESTAMPTZ NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### settings

Профили настроек runtime-приложения.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
environment TEXT NULL,
vars JSONB NOT NULL DEFAULT '[]'::jsonb,
auth JSONB NULL,
vocabs JSONB NOT NULL DEFAULT '[]'::jsonb,
sse JSONB NULL,
updates JSONB NOT NULL DEFAULT '[]'::jsonb,
custom_sections JSONB NOT NULL DEFAULT '[]'::jsonb,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Важно: настройки потенциально могут содержать секреты. На первом этапе можно хранить plain text, но код должен позволять позже добавить шифрование.

### types

Типы данных и ссылки на сущности.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
schema JSONB NOT NULL DEFAULT '{}'::jsonb,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
is_primitive BOOLEAN NOT NULL DEFAULT FALSE,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
author TEXT NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Seed-минимум:

```text
Any
ID
String
Number
Boolean
Null
DateTime
Time
RefAction
RefBinding
RefPresentationBinding
RefComponent
RefConverter
RefEnvironment
RefFilter
RefFolder
RefIntegration
RefNavigation
RefPage
RefPageTemplate
RefParameter
RefPolicy
RefProject
RefQuery
RefScenario
RefSettings
RefStyle
RefTenant
RefType
RefVocab
RefView
```

### queries

Описание REST/GraphQL/custom запросов.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
type TEXT NOT NULL,
endpoint TEXT NULL,
query TEXT NULL,
sub_field TEXT NULL,
method TEXT NULL,
headers JSONB NOT NULL DEFAULT '{}'::jsonb,
timeout_ms INTEGER NULL,
send_as_form_urlencoded BOOLEAN NOT NULL DEFAULT FALSE,
params JSONB NOT NULL DEFAULT '[]'::jsonb,
return_field JSONB NULL,
mock_data JSONB NULL,
mock_data_enabled BOOLEAN NOT NULL DEFAULT FALSE,
auth JSONB NULL,
filter_mode TEXT NULL,
filters JSONB NOT NULL DEFAULT '[]'::jsonb,
folder_id UUID NOT NULL REFERENCES folders(id),
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
author TEXT NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

`type`: `REST`, `GraphQL`, `Custom`.

### components

UI-компоненты и таблицы.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
component_type TEXT NOT NULL,
input_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
jsx_script TEXT NULL,
row_size TEXT NULL,
bindings JSONB NOT NULL DEFAULT '{}'::jsonb,
columns JSONB NOT NULL DEFAULT '[]'::jsonb,
schema JSONB NOT NULL DEFAULT '{}'::jsonb,
folder_id UUID NOT NULL REFERENCES folders(id),
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
author TEXT NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

`component_type`: `DSL`, `Table`.

### scenarios

Сценарии/скрипты.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
schema JSONB NOT NULL DEFAULT '{}'::jsonb,
folder_id UUID NOT NULL REFERENCES folders(id),
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
author TEXT NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### actions

Flow-действия.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
definition JSONB NOT NULL DEFAULT '{}'::jsonb,
input JSONB NULL,
output JSONB NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NOT NULL REFERENCES folders(id),
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
author TEXT NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Seed-минимум:

```text
console-log
load-vocabs
```

### parameters

Группы параметров.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
fields JSONB NOT NULL DEFAULT '[]'::jsonb,
active BOOLEAN NOT NULL DEFAULT TRUE,
folder_id UUID NOT NULL REFERENCES folders(id),
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
deleted_at TIMESTAMPTZ NULL,
author TEXT NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### filters

Фильтры для view/query/component.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
fields JSONB NOT NULL DEFAULT '[]'::jsonb,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
deleted_at TIMESTAMPTZ NULL,
author TEXT NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### converters

Переиспользуемые конвертеры данных.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Seed-минимум:

```text
iso-string-to-date
timestamp-to-date
date-to-iso-string
date-to-iso-z
string-to-date
date-to-date-string
date-to-time-string
time-string-to-date
iso-string-to-time-string
weekdays-range
string-trim
default-if-empty
string-to-boolean
to-array
split
string-to-number
number-to-string
json-parse
json-stringify
```

### integrations

Интеграции с внешними системами.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### views

Виды. Связывают компонент, запрос и фильтр.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
component_id UUID NULL REFERENCES components(id) ON DELETE SET NULL,
filter_id UUID NULL REFERENCES filters(id) ON DELETE SET NULL,
query_id UUID NULL REFERENCES queries(id) ON DELETE SET NULL,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### versions

Снапшоты конфигурации.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL,
description TEXT NULL,
data JSONB NOT NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

`identity` может повторяться, если нужно хранить несколько версий с одинаковым названием.

### environments

Окружения.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Seed:

```text
dev
prod
```

### tenants

Тенанты.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
code TEXT NOT NULL UNIQUE,
description TEXT NULL,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### policies

Политики.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### styles

Стили/темы.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
styles JSONB NOT NULL DEFAULT '{}'::jsonb,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Seed:

```text
default
```

### page_templates

Шаблоны страниц.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
areas JSONB NOT NULL DEFAULT '[]'::jsonb,
preview JSONB NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### pages

Страницы приложения.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
inherited BOOLEAN NOT NULL DEFAULT FALSE,
route_name TEXT NULL,
route_path TEXT NULL,
template_id UUID NULL REFERENCES page_templates(id) ON DELETE SET NULL,
controller_id UUID NULL REFERENCES views(id) ON DELETE SET NULL,
enabled BOOLEAN NOT NULL DEFAULT TRUE,
areas JSONB NOT NULL DEFAULT '[]'::jsonb,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### navigations

Деревья навигации.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
is_system BOOLEAN NOT NULL DEFAULT FALSE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
tree JSONB NOT NULL DEFAULT '[]'::jsonb,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### vocabs

Справочники.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
mode TEXT NOT NULL,
base_api_url TEXT NULL,
collection_slug TEXT NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
deleted_at TIMESTAMPTZ NULL,
meta JSONB NOT NULL DEFAULT '{}'::jsonb,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

`mode` минимум:

```text
external_payload
static
```

### i18n_bundles

Словари переводов.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
description TEXT NULL,
locales JSONB NOT NULL DEFAULT '{}'::jsonb,
active BOOLEAN NOT NULL DEFAULT TRUE,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
deleted_at TIMESTAMPTZ NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### behavior_bindings

Декларативные биндинги поведения.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
owner_type TEXT NOT NULL,
owner_id TEXT NOT NULL,
target_type TEXT NOT NULL,
target_id TEXT NOT NULL,
event_name TEXT NOT NULL,
script_ref TEXT NOT NULL,
mode TEXT NOT NULL DEFAULT 'replace',
priority INTEGER NOT NULL DEFAULT 0,
is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
environment_id UUID NULL REFERENCES environments(id) ON DELETE SET NULL,
is_inherited BOOLEAN NOT NULL DEFAULT FALSE,
origin_binding_id UUID NULL REFERENCES behavior_bindings(id) ON DELETE SET NULL,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

`mode`: `replace`, `append`, `prepend`, `disable`.

### presentation_bindings

Декларативные биндинги презентации.

```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
identity TEXT NOT NULL UNIQUE,
display_name TEXT NOT NULL,
project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
owner_type TEXT NOT NULL,
owner_id TEXT NOT NULL,
target_type TEXT NOT NULL,
target_id TEXT NULL,
role TEXT NOT NULL,
renderer_ref TEXT NOT NULL,
when_expression TEXT NULL,
mode TEXT NOT NULL DEFAULT 'replace',
priority INTEGER NOT NULL DEFAULT 0,
is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
environment_id UUID NULL REFERENCES environments(id) ON DELETE SET NULL,
is_inherited BOOLEAN NOT NULL DEFAULT FALSE,
origin_binding_id UUID NULL REFERENCES presentation_bindings(id) ON DELETE SET NULL,
folder_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

`mode`: `replace`, `append`, `prepend`, `disable`.

## Коллекции REST API

Для всех коллекций ниже нужно реализовать общий набор операций:

```text
projects
folders
settings
types
queries
components
scenarios
actions
parameters
filters
converters
integrations
views
versions
environments
tenants
policies
styles
page-templates
pages
navigations
vocabs
i18n-bundles
behavior-bindings
presentation-bindings
```

## REST API: чтение

### GET /api/{collection}

Получить список документов.

Query parameters:

```text
limit
offset
includeDeleted
onlyDeleted
sort
```

Правила:

- по умолчанию не возвращать `deleted_at IS NOT NULL`;
- если `includeDeleted=true`, вернуть активные и удаленные;
- если `onlyDeleted=true`, вернуть только удаленные;
- если коллекция не существует, вернуть `404`.

Ответ:

```json
{
  "docs": [],
  "total": 0,
  "limit": 50,
  "offset": 0
}
```

### GET /api/{collection}/{id}

Получить один документ по primary `id`.

Правила:

- `id` передается только в URL;
- `id` должен быть UUID;
- если документа нет, вернуть `404`;
- если документ мягко удален и не передан `includeDeleted=true`, вернуть `404`.

### POST /api/{collection}/lookup

Найти ровно один документ по уникальному ключу через body.

Body:

```json
{
  "id": "uuid optional",
  "identity": "identity optional",
  "includeDeleted": false
}
```

Правила:

- если передан `id`, искать по `id`;
- если передан `identity`, искать по `identity`;
- если переданы и `id`, и `identity`, они оба должны совпасть с одним документом;
- если не передан ни `id`, ни `identity`, вернуть `400 validation_error`;
- если документ не найден, вернуть `404 not_found`;
- если документ мягко удален и `includeDeleted=false`, вернуть `404 not_found`;
- endpoint возвращает один документ, а не список.

Ответ:

```json
{
  "doc": {}
}
```

### POST /api/{collection}/search

Поиск документов по фильтрам через body.

Body:

```json
{
  "id": "uuid optional",
  "identity": "identity optional",
  "projectId": "uuid optional",
  "folderId": "uuid optional",
  "includeDeleted": false,
  "onlyDeleted": false,
  "limit": 50,
  "offset": 0,
  "sort": [
    { "field": "createdAt", "direction": "desc" }
  ],
  "filters": {}
}
```

Правила:

- если body пустой, вернуть обычный список;
- если указан `id`, фильтровать по `id`;
- если указан `identity`, фильтровать по точному `identity`;
- если указаны `id` и `identity`, оба условия должны выполняться;
- если ничего не найдено, вернуть пустой список, не `404`;
- `includeDeleted` и `onlyDeleted` работают так же, как в `GET /api/{collection}`.

Ответ:

```json
{
  "docs": [],
  "total": 0,
  "limit": 50,
  "offset": 0
}
```

## REST API: запись

### POST /api/{collection}

Создать документ.

Правила:

- `POST /api/{collection}` используется только для создания, не для поиска;
- `identity` обязателен, если есть в таблице;
- `displayName` или `display_name` обязателен, если есть в таблице;
- при конфликте identity вернуть `409 conflict`;
- если передан несуществующий `folderId`, вернуть `400 validation_error`;
- если передан несуществующий `projectId`, вернуть `400 validation_error`.

Ответ:

- `201 Created`;
- тело созданного документа.

### PATCH /api/{collection}/{id}

Частично обновить документ.

Правила:

- нельзя обновить несуществующий документ: `404`;
- нельзя изменить `identity` на уже существующий в той же коллекции: `409`;
- нельзя напрямую менять `created_at`;
- `updated_at` обновляется автоматически;
- если обновляется relation на несуществующий id, вернуть `400`.

### DELETE /api/{collection}/{id}

Жестко удалить документ.

Правила:

- нельзя удалить несуществующий документ: `404`;
- нельзя удалить системный документ, если `is_system=true`: `409`;
- нельзя удалить root/system folder: `409`;
- если документ используется критичными ссылками, вернуть `409 referenced`;
- успешный ответ: `204 No Content`.

Важно: обычный `DELETE` должен быть жестким удалением. Для мягкого удаления есть отдельная операция.

## Доменные операции коллекций

### POST /api/{collection}/upsert

Создать или обновить документ по `identity`.

Body должен содержать `identity`.

Правила:

- если документа нет, создать;
- если документ есть, обновить;
- `identity` обязателен;
- ответ должен содержать `created`.

Ответ:

```json
{
  "created": false,
  "doc": {}
}
```

### PATCH /api/{collection}/{id}/folder

Переместить документ в другую папку.

Body:

```json
{
  "folderId": "uuid-or-null"
}
```

Правила:

- несуществующий документ: `404`;
- несуществующая папка: `400`;
- если `folderId=null`, документ переносится в корень секции;
- если тип папки не соответствует коллекции, вернуть `400`;
- для `folders` использовать отдельные правила папок.

### POST /api/{collection}/{id}/soft-delete

Мягко удалить документ.

Правила:

- несуществующий документ: `404`;
- уже мягко удаленный документ: вернуть `200`, операция идемпотентна;
- системный документ (`is_system=true`) нельзя мягко удалить: `409`;
- установить `deleted_at=NOW()`;
- при необходимости проставить `folder_id` на папку `soft-deleted`;
- вернуть обновленный документ.

### POST /api/{collection}/{id}/restore

Восстановить мягко удаленный документ.

Правила:

- несуществующий документ: `404`;
- если документ не удален, вернуть `200`, операция идемпотентна;
- установить `deleted_at=NULL`;
- если документ лежал в `soft-deleted`, перенести в корневую папку своей секции или `folder_id=NULL`;
- вернуть восстановленный документ.

### POST /api/{collection}/{id}/duplicate

Создать копию документа.

Body:

```json
{
  "identity": "new-identity",
  "displayName": "Copy name",
  "folderId": "uuid optional"
}
```

Правила:

- source не найден: `404`;
- новый `identity` уже занят: `409`;
- копия не должна наследовать `deleted_at`;
- копия должна попасть в ту же папку, если не передан `folderId`;
- `is_system` у копии должен быть `false`;
- вернуть `201 Created`.

## Операции с папками

### POST /api/folders

Создать папку.

Правила:

- `identity`, `displayName`, `entityType` обязательны;
- `identity` уникален;
- если указан `parentId`, parent должен существовать;
- `entityType` дочерней папки должен совпадать с `entityType` parent-папки, кроме системной папки `soft-deleted`.

### PATCH /api/folders/{id}

Обновить папку.

Запрещено:

- менять `identity` системной папки;
- менять `entityType` системной папки;
- делать parent равным самой папке;
- делать parent потомком текущей папки.

### POST /api/folders/{id}/soft-delete

Мягко удалить папку.

Правила:

- root/system folder удалить нельзя: `409`;
- папка переносится под `soft-deleted` или получает `deleted_at`;
- документы внутри этой папки считаются удаленными при построении дерева.

### POST /api/folders/{id}/restore

Восстановить папку.

Правила:

- если сохранен предыдущий parent, вернуть папку туда;
- иначе перенести в корень секции по `entityType`;
- если parent больше не существует, перенести в корень секции.

## Domain API

### GET /api/domain/export

Вернуть полный доменный dump.

Ответ:

```json
{
  "projects": [],
  "folders": [],
  "types": [],
  "queries": [],
  "components": [],
  "scenarios": [],
  "actions": [],
  "converters": [],
  "integrations": [],
  "views": [],
  "settings": [],
  "vocabs": [],
  "i18nBundles": [],
  "parameters": [],
  "filters": [],
  "versions": [],
  "environments": [],
  "tenants": [],
  "behaviorBindings": [],
  "presentationBindings": [],
  "policies": [],
  "styles": [],
  "pageTemplates": [],
  "pages": [],
  "navigations": []
}
```

Правила:

- по умолчанию не включать мягко удаленные документы;
- вернуть все коллекции, даже если они пустые.

### POST /api/domain/import

Импортировать полный dump.

Правила:

- на первом этапе допускается режим `upsert`;
- не удалять документы, которых нет в dump, если не передан `mode=replace`;
- проверять уникальность identity;
- все изменения выполнять в транзакции.

### POST /api/domain/compile

Проверить доменный dump на базовые ошибки.

Минимальные проверки:

- все referenced ids существуют;
- нет дублей `identity`;
- у pages есть существующий template, если `template_id` задан;
- у views есть существующие component/query/filter, если они заданы;
- folder references существуют.

## Versions API

### GET /api/versions

Список версий.

### POST /api/versions

Создать версию.

Body:

```json
{
  "identity": "2026-06-30-before-release",
  "description": "Before release",
  "projectId": "uuid-or-null",
  "data": {}
}
```

Если `data` не передан, сервис должен сам сделать `domain/export` и сохранить результат в `data`.

### POST /api/versions/{id}/restore

Восстановить dump из версии.

Правила:

- версия не найдена: `404`;
- восстановление выполнять в транзакции;
- по умолчанию использовать режим `upsert`, а не полную замену.

## Status codes

```text
200 OK                  successful read/update/action
201 Created             successful create/duplicate
204 No Content          successful hard delete
400 Bad Request         invalid input, invalid relation, invalid folder type
404 Not Found           collection/document/folder/version not found
409 Conflict            identity conflict, system delete, referenced document
422 Unprocessable Entity semantic validation error
500 Internal Error      unexpected error
```

## Ограничения и инварианты

1. Нельзя удалить несуществующий элемент.
2. Нельзя восстановить несуществующий элемент.
3. Нельзя переместить элемент в несуществующую папку.
4. Нельзя создать документ с duplicate identity.
5. Нельзя создать папку с parent, которого нет.
6. Нельзя создать цикл в дереве папок.
7. Нельзя жестко удалить системные root-папки.
8. Нельзя по умолчанию отдавать мягко удаленные сущности в обычных списках.
9. Нельзя сохранять невалидный JSON в JSONB-поля.
10. Нельзя менять `created_at` через API.
11. `POST /api/{collection}` нельзя использовать как поиск; это только создание.
12. `POST /api/{collection}/lookup` должен возвращать один документ или ошибку.
13. `POST /api/{collection}/search` должен возвращать список, даже если найден 0 или 1 документ.
14. Операции import/restore версии должны выполняться в транзакции.

## Минимальные acceptance criteria первого этапа

1. Миграции создают все таблицы.
2. Seed создает системные папки, базовые типы, базовые окружения, базовый стиль, базовые конвертеры и базовые actions.
3. `go test ./...` проходит.
4. `GET /api/domain/export` возвращает все коллекции.
5. Для каждой коллекции работает:
   - `GET /api/{collection}`;
   - `GET /api/{collection}/{id}`;
   - `POST /api/{collection}/lookup`;
   - `POST /api/{collection}/search`;
   - `POST /api/{collection}`;
   - `PATCH /api/{collection}/{id}`;
   - `DELETE /api/{collection}/{id}`;
   - `POST /api/{collection}/upsert`;
   - `POST /api/{collection}/{id}/soft-delete`;
   - `POST /api/{collection}/{id}/restore`.
6. Проверены ошибки:
   - lookup без `id` и `identity` -> `400`;
   - lookup unknown -> `404`;
   - search unknown -> `200` с пустым `docs`;
   - delete unknown -> `404`;
   - patch unknown -> `404`;
   - create duplicate identity -> `409`;
   - move to unknown folder -> `400`;
   - delete system folder -> `409`.
7. OpenAPI YAML описывает все реализованные endpoints.

## Что не входит в первый этап

- Авторизация и роли.
- WebSocket/SSE updates.
- Redpanda/Kafka.
- OpenTelemetry collector.
- Полная совместимость с Payload Admin UI.
- UI для администрирования.
- Шифрование секретов.
- Сложная компиляция runtime-домена.
- Полная нормализация всех вложенных JSON-структур в отдельные таблицы.

## Рекомендация по реализации

Начать с универсального слоя для CRUD-коллекций:

```text
internal/domain/entities
internal/ports
internal/repo/postgres
internal/usecase
internal/api/http
```

Для таблиц со схожей структурой можно сделать общий repository/helper, но не нужно строить слишком абстрактный ORM. Главное - ясные миграции, стабильный REST-контракт и предсказуемая бизнес-логика.
