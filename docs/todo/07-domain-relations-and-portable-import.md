# 07. Domain Relations and Portable Import — история изменений

## Статус

Выполнено. Исходная постановка задачи находится в
`../task/07-domain-relations-and-portable-import (DONE).md`.

## Реализовано

### Relation resolver

- Создан transport- и PostgreSQL-независимый package
  `internal/usecase/relations`.
- `Resolver` разрешает Project и Folder по public identity только в workspace
  из `context.Context`, нормализует input и проверяет entity type и project
  scope папки.
- Queries, Data Views, Converters и Tenants используют общий resolver; legacy
  Components намеренно не переносились, потому что удаляются в задаче 20.
- HTTP handler не выполняет relation lookup и не принимает raw foreign UUID.

### Dependency index и delete guard

- Добавлены `domain_dependency_states` и `domain_dependencies`: owner state
  хранится даже для пустого, но непроверенного документа; dependency rows
  каскадно удаляются вместе с state.
- Схема добавлена отдельной migration
  `000022_init_domain_dependencies.sql`. Следующая и последняя migration
  `000024_seed_swagger_demo_data.sql` содержит только demo/Swagger data.
- Repository и usecase поддерживают replace, delete, usages и
  `EnsureNotReferenced`; update projection выполняется в transaction context.
- Dependency extractor типизирован, нормализует и дедуплицирует references,
  сохраняя `verified` или `unverified` state.
- При ссылках на удаляемую identity формируется domain conflict
  `<entity>_in_use` с первыми 20 usages и полным total.

### Usages API

Добавлен scoped read-only endpoint:

```text
GET /api/v1/domain/usages
```

- Обязательны `dependency_type` и `dependency_identity`.
- Поддержаны `limit` (50 по умолчанию, максимум 200) и `offset`.
- Ответ возвращает owner type/identity/UUID, source path, verification state и
  total; пустой результат остаётся корректным `200` с пустым списком.
- Endpoint подключён в Fx, маршруты и generated OpenAPI. Public API не даёт
  изменять derived projection.

### Portable import/export foundation

- Добавлен `internal/usecase/portable` с contracts для
  `PortableDocument`, `PortableRelation`, `ImportOptions`, `ImportResult` и
  `EntityPortableAdapter`.
- Registry и planner работают с portable identities, а target UUID строят лишь
  внутри import graph. Canonical JSON не переписывается.
- Поддержаны conflict policies `fail`, `overwrite` и только явный `rename`.
- Non-atomic import сохраняет валидные документы и сообщает document errors;
  atomic import исполняется через transaction manager и не оставляет partial
  graph.

### Database integrity

Согласно актуальному соглашению проекта, structural изменения внесены в
существующие schema migrations, а не наложены поверх них. Project, Folder и
связанные domain tables получили составные workspace foreign keys, чтобы
cross-workspace relation не могла попасть в базу. Чистая PostgreSQL-БД
накатила migrations 001–023 без ошибок порядка.

## Проверки

Пройдены:

```bash
GOCACHE=/tmp/service-backend-go-build go test ./...
GOCACHE=/tmp/service-backend-go-build go test -race ./...
TEST_POSTGRES_DSN=... go test -tags=integration \
  ./internal/repo/postgres -run '^TestDomainDependenciesRepositoryReplaceDeleteAndRollback$' -count=1
git diff --check
```

Живой PostgreSQL-прогон проверил накатывание всех migrations, replace/delete
dependency projection, rollback transaction, usage lookup, индексы, unique
constraint и cascade owner state. Полный tagged integration package отдельно
содержит два pre-existing падающих ожидания: тест ожидает пустую БД после
demo-seed, а другой ожидает `not_found` вместо существующего
`tenant_not_found`; целевой тест задачи 07 проходит.

## Итоговый flow

```text
public identity
  -> shared relation resolver (workspace context)
  -> entity usecase + typed dependency extractor
  -> transaction
  -> dependency state / dependency rows
  -> usages API или delete guard
```
