# Ручная проверка release cache через Scalar

Этот сценарий проверяет HTTP-контракт release и cache без необходимости вручную
собирать или переносить большой portable snapshot. Между запросами понадобятся
только UUID существующего commit и ETag из HTTP-ответа.

## Подготовка

В `.env.local` сейчас настроен OIDC, поэтому Scalar без bearer token получает
`401` — это нормальное поведение. Для проверки release под дебаггером не меняйте
этот файл и не отключайте OIDC в нём.

Создайте в GoLand отдельную конфигурацию Run/Debug, например
`service-backend (release-cache)`. Её **Environment variables**:

```text
APP_ENV=local;AUTH_MODE=dev;AUTH_LOGIN_ADAPTER=dev
```

Рабочая папка — `service-backend`. Переменные из Run/Debug-конфигурации уже
присутствуют в процессе, поэтому `.env.local` не перезапишет их. `APP_ENV=local`
нужен для выбора `configs/local.yaml` ещё до загрузки env-файла.

Запустите эту конфигурацию под Debug и откройте
[`http://localhost:8080/swagger`](http://localhost:8080/swagger). В этом
временном dev-режиме bearer token не требуется.

Во все запросы ниже добавьте header:

```http
X-Endge-Workspace: default
```

## Сценарий

1. В разделе **Коммиты** вызовите `GET /api/v1/commits`.

   Bootstrap создаёт начальный commit. Скопируйте значение `items[0].id` из
   ответа. Это единственное значение, которое потребуется вставить в JSON.

2. В разделе **Релизы** вызовите `POST /api/v1/releases` с этим телом.
   Замените только `<commit-id>` на скопированный UUID.

   ```json
   {
     "identity": "cache-check-local",
     "displayName": "Проверка release cache",
     "description": "Ручная проверка через Scalar",
     "sourceCommitId": "<commit-id>"
   }
   ```

   Если сценарий запускается повторно, измените `identity`: он уникален внутри
   workspace.

3. Вызовите `GET /api/v1/releases/cache-check-local`.

   В ответе находятся только metadata release. Поля `data` и большого JSON
   snapshot там намеренно нет.

4. Дважды вызовите `GET /api/v1/releases/cache-check-local/export` без
   `If-None-Match`.

   Оба ответа имеют статус `200` и возвращают одинаковый portable snapshot.
   Второй запрос получает artifact из process-local cache. Этот большой JSON не
   нужно сохранять или вставлять в другой endpoint.

5. Скопируйте response header `ETag` из одного export-ответа, включая кавычки.
   Повторите тот же `GET`, добавив header:

   ```http
   If-None-Match: "<etag-from-response>"
   ```

   Scalar должен показать `304 Not Modified` без response body. Авторизация
   workspace выполняется до проверки ETag и ответа `304`.

6. Необязательно: создайте второй release с новой `identity`, затем вызовите
   `GET /api/v1/releases/last/export`.

   Ответ должен относиться к последнему созданному release, а не к старому
   cache entry.

## Что проверяется автоматически

Scalar проверяет HTTP-контракт вручную. Дедупликация десяти одновременных
запросов, LRU-вытеснение и защита кешируемых bytes проверяются unit-тестом:

```bash
go test ./internal/usecase/release_artifacts -count=1
```
