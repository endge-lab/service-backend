# 13. Vocabs Foundation

## Цель

Реализовать `RVocabs` — конфигурацию внутреннего или внешнего vocabulary provider. Этот сервис хранит descriptor и не проксирует данные внешнего словаря.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля

```text
id UUID, workspace_id UUID, identity TEXT, display_name TEXT,
description TEXT NULL, mode TEXT, base_api_url TEXT NULL,
collection_slug TEXT NULL, auth_mode TEXT DEFAULT 'inherit',
auth_profile_id UUID NULL, folder_id UUID NULL,
active BOOLEAN, deleted_at TIMESTAMPTZ NULL, meta JSONB,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Allowed values:

```text
mode: internal | external_payload
authMode: inherit | profile | manual | none
```

For `external_payload`, `baseApiUrl` and `collectionSlug` are required. For `authMode=profile`, `authProfileIdentity` обязателен и должен существовать в текущем workspace. HTTP принимает identity, usecase разрешает его по `(workspace_id, identity)`, а в таблице сохраняется только `auth_profile_id UUID`. Сырые credentials и tokens не хранить. Uniqueness: `(workspace_id, identity)`.

Так как `auth_profiles` создаётся в задаче №18, текущая migration добавляет nullable `auth_profile_id`, а migration задачи №18 завершает relation: composite foreign key `(workspace_id, auth_profile_id) -> auth_profiles(workspace_id, id)`, `ON DELETE RESTRICT` и constraint `auth_mode = 'profile'` только при непустом `auth_profile_id`. Для остальных auth modes поле должно быть `NULL`.

Добавить folder type `vocabs` и root `root-vocabs`.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/vocabs?mode=external_payload&active=true&include_deleted=false` — list; filters optional.
- `POST /api/v1/vocabs` — создать descriptor.
- `GET /api/v1/vocabs/:vocab_ref` — получить по UUID или identity.
- `PATCH /api/v1/vocabs/:vocab_id` — обновить по UUID.
- `DELETE /api/v1/vocabs/:vocab_id` — soft-delete.
- `POST /api/v1/vocabs/:vocab_id/restore` — restore.
- `DELETE /api/v1/vocabs/:vocab_id/hard` — hard-delete, если descriptor не используется.

Create request:

```json
{
  "identity": "airports",
  "displayName": "Airports",
  "description": "Remote airport vocabulary",
  "mode": "external_payload",
  "baseApiUrl": "https://data.example.test/api",
  "collectionSlug": "airports",
  "authMode": "profile",
  "authProfileIdentity": "data-api",
  "folderIdentity": "root-vocabs",
  "active": true,
  "meta": {}
}
```

Response возвращает те же fields плюс UUID/timestamps/deletedAt. PATCH partial; identity immutable. Backend может проверить URL format, но не обязан выполнять network request.

Errors: `vocab_not_found`, `vocab_identity_conflict`, `vocab_in_use`, `external_vocab_config_required`, `auth_profile_not_found`, folder errors.

## Acceptance Criteria

Conditional validation, workspace scoping, lifecycle, OpenAPI/tests готовы; identity relation не хранится как `TEXT`; FK и conditional constraint завершаются в задаче №18; secrets не принимаются и не логируются; `go test ./...` проходит.
