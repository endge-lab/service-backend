# Спецификация тестирования: template architecture

## Контекст
- `service-template` — это эталон нового Endge-сервиса, поэтому здесь важно тестировать не только reference-feature, но и сам архитектурный стандарт.

## Границы
- структура папок `internal/*`
- package naming слоёв
- dependency boundaries между `domain`, `services`, `usecase`, `api/http`, `repo/postgres`
- reference wiring в `internal/bootstrap/usecase.go`
- обязательные артефакты `docs/openapi.yaml` и `test/*`

## Риски
- `P0`: шаблон перестаёт быть эталоном и новые сервисы стартуют с размытыми слоями
- `P1`: package naming или wiring уходят от стандарта, но сервис ещё собирается
- `P2`: documentation/test scaffolding расходятся с реальным устройством шаблона

## Матрица покрытия

| Сценарий | Критичность | Architecture Test | Unit | Status | Артефакты | Комментарий |
|---|---|---|---|---|---|---|
| Required folders и checked-in артефакты шаблона существуют | P0 | done | n/a | done | `test/architecture/architecture_test.go` | Защищает минимальный скелет шаблона |
| Имена package совпадают со слоями template (`usecase`, `services`, `ports`, `repo/postgres`, `api/http`) | P0 | done | n/a | done | `test/architecture/architecture_test.go` | Не даёт вернуться к старым именам и смешанным пакетам |
| `domain` не зависит от HTTP, Postgres и middleware | P0 | done | n/a | done | `test/architecture/architecture_test.go` | Базовый инвариант clean architecture |
| `usecase` и `services` не зависят напрямую от `repo/postgres` и transport | P0 | done | n/a | done | `test/architecture/architecture_test.go` | Защищает application layer от деградации |
| `api/http` не ходит напрямую в `repo/postgres`, а `repo/postgres` не зависит от transport | P1 | done | n/a | done | `test/architecture/architecture_test.go` | Контролирует boundary между transport и persistence |
| `bootstrap/usecase.go` собирает reference-flow через `ports`, `services`, `usecase`, `repo/postgres`, `api/http` | P1 | done | n/a | done | `test/architecture/architecture_test.go` | Проверяет эталонную DI-сборку |
| `RedPanda` runtime подключается через `internal/platform` и env-конфиг template | P1 | done | done | done | `internal/platform/redpanda.go`, `internal/config/config.go`, `internal/platform/redpanda_test.go` | Не даёт дублировать Kafka bootstrap в каждом новом сервисе |
| README и docs/tests фиксируют архитектурные правила и тестовую стратегию | P2 | n/a | n/a | done | `README.md`, `docs/architecture.md`, `docs/tests/*` | Документация синхронизирована с автоматическими проверками |

## Пробелы
- Архитектурные тесты сейчас проверяют статическую структуру и imports, но не валидируют все допустимые/недопустимые типовые зависимости на уровне `go list` graph.
- Если template расширится новыми reference-модулями, матрицу и tests нужно обновлять вместе с ними, а не постфактум.
