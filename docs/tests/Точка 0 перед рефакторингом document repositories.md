# Точка 0 перед рефакторингом document repositories

Дата фиксации: 2026-09-06 15:56 +03:00.

## Состояние исходного кода

- `HEAD`: `2739ca6576ee820fd063a5633c4f9f77c87b028f`.
- Рабочее дерево намеренно не чистое: в нём находятся изменения задачи release
  artifact cache и текущая нормализация lint. Точка 0 относится именно к этому
  рабочему дереву до начала устранения `dupl` в PostgreSQL document adapters.
- `git diff --check` завершился успешно.

## Результаты проверок

| Команда | Результат |
|---|---|
| `go test ./...` | успешно |
| `make test-unit` | успешно |
| `make test-integration` | успешно, 9.274 s |
| `make test-e2e` | успешно, 9.505 s |

`make test-unit`, `make test-integration` и `make test-e2e` составляют
`make test-critical`. Integration и E2E создают изолированные временные
PostgreSQL-контейнеры, применяют миграции и удаляют контейнеры после прогона.

## Что защищает точка 0

Для рефакторинга document repositories наиболее важен
`TestAllDocumentHTTPContracts` из `test/e2e/http_test.go`: он проходит полный
HTTP lifecycle всех 23 document collections — list, create, get, patch с ETag,
delete и restore — через настоящий PostgreSQL.

Полная карта текущего HTTP-покрытия находится в
[Карте E2E покрытия API](../Карта%20E2E%20покрытия%20API.md). После расширения
`service_api_test.go`, `access_grants_test.go`, `release_read_test.go` и
`configurator_auth_test.go`, `revision_detail_test.go` и `ai_api_test.go` E2E
покрывают 210 из 210 OpenAPI операций. Полная точка 0 на весь API достигнута.

## Расширение точки 0: service API

После первоначальной фиксации добавлены и успешно выполнены E2E-сценарии для
`GET /version`, поиска service users, list/get/patch/list-members workspaces,
полного lifecycle integrations и list/create/delete backend connections.
Они проверяют authentication, RBAC, workspace isolation, ETag/conflict и
soft-delete/restore там, где эти свойства входят в контракт.

Команда `go test -tags=e2e -count=1 ./test/e2e` завершилась успешно за 12.676 s.

Последнее расширение добавило lifecycle workspace/platform grants и bulk grants
для `selected` и `all-active`: сценарии проверяют немедленное предоставление и
отзыв реального доступа к защищённым HTTP-операциям. E2E-прогон завершился
успешно за 15.184 s. Полный `make test-critical` после этого расширения:
integration — 8.007 s, E2E — 16.240 s.

Последнее расширение закрыло list/get releases и полный public browser-auth
flow: login redirect с PKCE/state/nonce, callback с code exchange, server-side
session и logout с локальным revoke. E2E-прогон завершился успешно за 16.111 s.
Полный `make test-critical` после этого расширения: integration — 8.070 s,
E2E — 17.773 s.

Финальное расширение добавило revision detail и все AI API операции. AI E2E
используют локальный fake Workbench, но проходят настоящий gRPC adapter, health
check и SSE Run; credential дополнительно проверяется на encrypted хранение.
E2E-прогон завершился успешно за 18.186 s.
Финальный `make test-critical`: integration — 8.038 s, E2E — 17.547 s.

## Правило сравнения после рефакторинга

После каждого законченного этапа рефакторинга нужно повторить проверки из
таблицы. Любой новый fail, изменение HTTP status, ETag, данных или прав доступа
считается регрессией и должен быть исправлен до следующего этапа.
