# 21. Детальная observability use-case слоя

## Цель

Каждый завершённый значимый шаг use case должен быть виден одновременно в
trace и в структурированных логах. Это позволяет по `trace_id` восстановить
путь операции в OpenSearch и понять, на каком бизнес-шаге она остановилась.

Полный reference-поток находится рядом:
[`21-usecase-observability.example.md`](21-usecase-observability.example.md).

## Граница задачи

- Меняем только `internal/usecase` и общий helper в
  `internal/observability`.
- HTTP сохраняет входящий trace context и не дублирует бизнес-события.
- Repository не получает отдельные логи или spans на каждый SQL-вызов.
- Existing recorder продолжает записывать одну metric на завершённую
  use-case operation; identity и UUID не становятся metric labels.

## Правило одного шага

После каждого **успешно завершённого бизнес-шага** use case вызывает
`observed.RecordStep`. Метод добавляет OTel span event и пишет `Info` log с
тем же смыслом и коррелирующим `trace_id`.

К таким шагам относятся:

1. input нормализован и провалидирован;
2. родительская или целевая сущность разрешена;
3. проверка существования/предусловия завершена;
4. сущность создана, обновлена, удалена, восстановлена или получила новое
   состояние;
5. набор данных или итоговый результат получен.

Не создаём события для чистых присваиваний, преобразования структур или
вызовов private helper, которые не обозначают отдельное завершённое
бизнес-действие.

## Именование

Формат event name: `<entity>.<operation>.<completed_step>`.

Примеры:

- `converter.create.input_validated`
- `converter.create.project_resolved`
- `converter.create.identity_available`
- `converter.create.persisted`
- `query.list.result_loaded`

Сообщение `Info` лога формулируется в прошедшем времени: `converter create
input validated`, `project resolved for converter create`, `converter
persisted`.

## Ошибки и поля

- Ошибка шага остаётся на текущем error-path: `logOperationError` пишет
  `Warn` для ожидаемых 4xx и `Error` для инфраструктурных 5xx.
- `defer observed.End(&err)` завершает span и operation metric с финальной
  ошибкой.
- В events и логах допустимы технические ID, identity, entity type и count.
- Нельзя писать пароли, токены, исходные payload/configuration и персональные
  данные. Нельзя переносить ID/identity в labels метрик.

## Порядок внедрения

1. Добавить и проверить `Operation.RecordStep`.
2. Внедрить pattern в create/update/get/list/count/state methods модулей
   `converters`, `projects`, `folders`, `queries`, `data_views`, `workspaces`,
   `session`, `components_legacy`.
3. Заменить прежние финальные success `Debug` на `RecordStep`/`Info` и не
   дублировать одинаковые записи.
4. Тестами зафиксировать, что важный happy path создаёт ожидаемые span events
   и structured Info logs.
