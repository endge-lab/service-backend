# CHANGELOG

## 0.3.6 — 2026-08-25

### Добавлено

- Безопасный список разрешённых origins для динамического возврата в локальный или развёрнутый Configurator после OIDC-входа.

## 0.3.0 — 2026-08-24

### Добавлено

- Source-first контракт Actions с обязательными `source` и `sourceVersion`.
- Безопасная миграция пустых legacy Flow документов и явная остановка на непустых Flow.

### Breaking changes

- Поля `definition`, `input` и `output` удалены из Action HTTP contract.

## 0.2.0 — 2026-08-04

### Добавлено

- Универсальная production-аутентификация через OIDC/JWKS, совместимая с Keycloak и другими OIDC-провайдерами.
- Автоматическое создание и обновление локальной проекции пользователя при первом аутентифицированном запросе.
- Workspace RBAC с ролями `viewer`, `editor`, `admin`, Platform Admin и implicit-доступом `editor` к workspace `default`.
- Полный CRUD согласованных MVP-документов с ETag/`If-Match`, soft-delete, restore и optimistic concurrency control.
- Полные document revisions, атомарные mutation batches и восстановление документа через добавление новой revision.
- Workspace commits с режимами `preserve` и optional `squash`, contributors и восстановлением полного состояния workspace.
- Immutable releases, привязанные к commit и содержащие готовый portable JSON snapshot без replay истории.
- Единый live workspace snapshot для загрузки Configurator одним запросом.
- Двухфазный безопасный import (`plan` + подтверждение с `If-Match`), который применяет полный snapshot как новые revisions и один обратимый import commit, сохраняя предыдущую историю.
- Автоматические `pre_import` backups, ручные backups с описанием, экспорт последнего backup и ZIP-архив всех страховочных копий.
- Inline JSON export по умолчанию и опциональная выдача attachment через `download=true`; read-only alias `last` для последнего release.
- Типизированные composite FK по `workspace_id` для `Update.store` и `Vocab.authProfile`, а также отдельная many-to-many таблица `project_environments`.
- System root folders, защита от циклов и перенос содержимого при soft-delete папки.
- Декларативная transport validation через `service-kit-go` с единым `validation_error.details.fields` и запретом неизвестных JSON-полей.
- Focused contract tests для strict decoding, validation, ETag и обязательного `If-Match`.
- Versioned encryption keyring для безопасной ротации refresh token и временных OIDC-данных без принудительного завершения активных сессий.
- Lifecycle-очистка просроченных login transactions и browser sessions.

### Изменено

- Полностью переписана миграционная цепочка для развёртывания на пустой PostgreSQL.
- Документные HTTP handlers, usecases, repository ports и PostgreSQL adapters разделены по ресурсам; общими оставлены только transaction/history механизмы.
- Каждый HTTP resource разделён на `handler.go`, `routes.go`, `transport.go` и `usecase.go`; универсальный `shared.DocumentHandler` удалён, request/response DTO принадлежат конкретным ресурсам.
- Монолитные packages `internal/usecase/mvp` и `internal/api/http/v1/mvp` удалены, а агрегат `entities/mvp.go` разделён на доменные сущности по назначению.
- Production теперь запускается только с полной OIDC-конфигурацией; `AUTH_MODE=dev` разрешён только вне production.
- Query переведён на source-first контракт с обязательным `sourceVersion=2`.
- Из Project удалена неиспользуемая связь с Navigation.
- Contributors сжатой revision хранятся непосредственно в `document_revisions.contributor_user_ids` без отдельной relation-таблицы.
- Canonical Component хранится только в SFC-формате; старый `componentSFCs` нормализуется при portable import.
- OpenAPI и документация запуска обновлены под новый backend-контракт.
- Generic Scalar paths заменены на отдельные CRUD paths и типизированные request/response schemas всех 22 MVP-коллекций.
- HTTP handlers переведены на локальные `UseCase` interfaces, application write inputs стали типизированными, а panic-based response mapping удалён.
- Browser sessions больше не сохраняют access token; обычная проверка сессии не блокирует строку PostgreSQL, а `FOR UPDATE` используется только во время refresh identity.

### Breaking changes

- База должна быть создана заново миграциями `000001`–`000039`.
- Старый Payload API напрямую не совместим с новым контрактом; frontend должен использовать adapter.
- Весь `/api` требует Current User, а workspace-scoped endpoints — заголовок `X-Endge-Workspace`.
- PATCH, DELETE и restore требуют актуальный `If-Match`.

## 0.1.0

- Подготовлен публичный Go module `github.com/endge-lab/service-backend`.
- Шаблон переведен на зависимость `github.com/endge-lab/service-kit-go v0.1.0`.
- Удален `package.json`: версия Go-сервиса задается git tag-ами и `CHANGELOG.md`.
- Добавлена инструкция на русском по публикации, локальному `go.work` и подключению kit из локальной папки.
- Auth сделан опциональным через `AUTH_ENABLED=false` по умолчанию.
- Telemetry сделана опциональной через `TELEMETRY_ENABLED=false` по умолчанию.
- Redpanda/Kafka оставлены optional и выключены по умолчанию.
- Реальные `.env.*` заменены на безопасные `.env.*.example`.
- CI заменен на GitHub Actions.
- Dockerfile и docker-compose очищены от приватных module credentials.
