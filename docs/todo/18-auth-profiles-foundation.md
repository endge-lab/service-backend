# 18. Auth Profiles Foundation

## Цель

Реализовать `RAuthProfile` — конфигурацию client-side auth adapter и references на credentials. Это не backend user authentication: сервис остаётся открытым и не выполняет login flow.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля

```text
id UUID, workspace_id UUID, identity TEXT, display_name TEXT,
description TEXT NULL, adapter_id TEXT, config JSONB,
credential_refs JSONB, persist TEXT, folder_id UUID NULL,
active BOOLEAN, deleted_at TIMESTAMPTZ NULL, meta JSONB,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Allowed values:

```text
adapterId: keycloak_manual | keycloak_form | manual_token
persist: localStorage | sessionStorage | memory
```

`config` — adapter-specific JSON. `credentialRefs` — map string keys to workspace variable/env/secret references. Raw passwords, access tokens and client secrets запрещены. Uniqueness `(workspace_id, identity)`.

Добавить `UNIQUE (workspace_id, id)`, чтобы другие workspace-scoped tables могли использовать composite foreign keys и не ссылались на профиль из другого workspace.

## Завершение auth profile relations

Эта migration должна завершить relations, колонки для которых созданы раньше:

```text
(workspaces.id, workspaces.default_auth_profile_id)
  -> auth_profiles(workspace_id, id) ON DELETE RESTRICT

(workspaces.id, workspaces.sse_auth_profile_id)
  -> auth_profiles(workspace_id, id) ON DELETE RESTRICT

(vocabs.workspace_id, vocabs.auth_profile_id)
  -> auth_profiles(workspace_id, id) ON DELETE RESTRICT
```

Также добавить conditional constraints:

- `workspaces.sse->>'authMode' = 'profile'` требует `sse_auth_profile_id IS NOT NULL`;
- для другого SSE auth mode `sse_auth_profile_id IS NULL`;
- `vocabs.auth_mode = 'profile'` требует `auth_profile_id IS NOT NULL`;
- для другого vocab auth mode `auth_profile_id IS NULL`.

HTTP продолжает работать с `defaultAuthProfileIdentity`, `sse.authProfileIdentity` и `authProfileIdentity`. Usecases разрешают identity в UUID внутри текущего workspace. При soft-delete и hard-delete profile проверить все эти dependencies и вернуть `auth_profile_in_use`; сначала клиент должен очистить или заменить ссылки.

Добавить folder type `auth-profiles` и root `root-auth-profiles`.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/auth-profiles?adapter_id=manual_token&active=true&include_deleted=false` — list.
- `POST /api/v1/auth-profiles` — создать profile.
- `GET /api/v1/auth-profiles/:auth_profile_ref` — получить по UUID или identity.
- `PATCH /api/v1/auth-profiles/:auth_profile_id` — partial update по UUID.
- `DELETE /api/v1/auth-profiles/:auth_profile_id` — soft-delete.
- `POST /api/v1/auth-profiles/:auth_profile_id/restore` — restore.
- `DELETE /api/v1/auth-profiles/:auth_profile_id/hard` — hard-delete, если profile не referenced.

Create request:

```json
{
  "identity": "main-keycloak",
  "displayName": "Main Keycloak",
  "description": "Primary auth profile",
  "adapterId": "keycloak_form",
  "config": {
    "baseUrl": "{KEYCLOAK_URL}",
    "realm": "endge"
  },
  "credentialRefs": {
    "clientId": "workspace:KEYCLOAK_CLIENT_ID",
    "clientSecret": "secret:KEYCLOAK_CLIENT_SECRET"
  },
  "persist": "sessionStorage",
  "folderIdentity": "root-auth-profiles",
  "active": true,
  "meta": {}
}
```

Response не разворачивает credential references и не содержит secret values. PATCH принимает все editable fields кроме identity. Logs/errors должны redact config keys, помеченные как sensitive.

Errors: `auth_profile_not_found`, `auth_profile_identity_conflict`, `auth_profile_in_use`, `invalid_adapter_id`, `invalid_persist_mode`, `raw_credential_forbidden`, folder errors.

## Acceptance Criteria

Open backend auth не добавлен; secret values не persisted/returned/logged; structured relations сохраняются как UUID и защищены workspace-scoped FK/constraints; references валидируются структурно; lifecycle/OpenAPI/tests готовы; `go test ./...` проходит.
