# Спецификации тестирования

Эта папка описывает не только функциональные тесты шаблона, но и то, как мы проверяем сам архитектурный стандарт.

Для `service-template` тестовая спецификация нужна в двух ролях:

- показать, что reference-feature и transport/repo контур работают;
- зафиксировать, что структура шаблона не расползается и новые сервисы стартуют из корректной архитектуры.

## Какие уровни проверок ожидает template

- `unit` для `domain`, `services`, `usecase`
- `architecture` для структуры слоёв, package naming и dependency boundaries
- `integration` для реальной БД и repo adapters
- `contract` для HTTP-контракта
- `e2e` для полного пользовательского потока

Для event-driven сервисов поверх template дополнительно ожидаются:

- unit tests на валидацию и нормализацию Kafka/RedPanda payload;
- integration tests на batch-delivery и persistence;
- contract tests на polling/receipt endpoints;
- e2e tests на полный цикл ingest -> delivery -> client acknowledgement.

## Как считать покрытие по template

У template есть два независимых типа покрытия:

- техническое покрытие кода через `go test -coverpkg=./...`
- покрытие архитектурных сценариев через матрицу в отдельных markdown-файлах

Архитектурный сценарий считается закрытым, если для него есть:

- формализованное правило в `README` или `docs/architecture.md`
- автоматическая проверка в `go test`
- ссылка на конкретный тестовый артефакт в матрице покрытия

## Что именно стоит фиксировать в спецификациях

- required folders и naming слоёв
- запрет недопустимых зависимостей между слоями
- reference wiring в `bootstrap/usecase.go`
- наличие checked-in `OpenAPI`
- наличие test scaffolding `integration / contract / e2e`

## Файлы в этой папке

- `template-architecture.md` — матрица покрытия архитектурных правил шаблона

## Обязательный smoke для shared runtime

- После изменения middleware, telemetry, logging, Redpanda wiring или auth runtime обязательно прогоняем `go test ./...` на сервисе целиком.
- Если сервис использует `service-kit-go`, дополнительно проверяем, что сборка проходит и в локальном `go.work`, и в CI-режиме с приватным Go module из GitHub.
- Для publish/deploy сценариев smoke-check включает `docker compose config` и успешный `go mod download` c переменными `GO_PRIVATE_MODULES_HOST`, `GO_PRIVATE_MODULES_PATTERN`, `GO_PRIVATE_MODULES_TOKEN`.
