# 17. Filters Foundation

## Цель

Реализовать `RFilter`: independent source-first filter definition плюс legacy-compatible `fields`. Эти два authoring contracts хранятся независимо и не генерируют друг друга.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля

```text
id UUID, workspace_id UUID, folder_id UUID NULL,
identity TEXT, display_name TEXT, fields JSONB DEFAULT [],
source TEXT DEFAULT '', source_version INTEGER DEFAULT 1,
active BOOLEAN, deleted_at TIMESTAMPTZ NULL,
meta JSONB, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Filter field:

```json
{
  "key": "airlines",
  "label": "Airlines",
  "mode": "vocab",
  "staticOptions": [],
  "vocabIdentity": "airlines",
  "vocabCollection": "airlines",
  "valuePath": "code",
  "displayNamePath": "displayName",
  "multiple": true,
  "converterIdentities": ["trim"],
  "defaultValue": "*",
  "active": true
}
```

Allowed modes: `static | vocab | date | time | datetime | boolean | string | number`. `key` unique inside filter. Static mode requires options; vocab mode requires vocab fields and existing `vocabIdentity`. Converter identities resolve inside workspace.

`vocabIdentity` и `converterIdentities` остаются identities внутри portable authoring JSON, не заменяются UUID и попадают в derived dependency index из задачи №7. Backend использует index для usages и запрета hard-delete vocabulary/converter; если source parser ещё недоступен, состояние source dependencies помечается `unverified`. Uniqueness `(workspace_id, identity)`, `sourceVersion >= 1`.

Добавить folder type `filters` и root `root-filters`; folder должна принадлежать текущему workspace. Filter не содержит `project_id`.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/filters?folder_identity=...&include_deleted=false` — summaries без fields/source.
- `POST /api/v1/filters` — создать filter.
- `GET /api/v1/filters/:filter_ref` — detail по UUID или identity.
- `PATCH /api/v1/filters/:filter_id` — metadata и полный `fields` array update по UUID.
- `PUT /api/v1/filters/:filter_id/source` — заменить independent source.
- `DELETE /api/v1/filters/:filter_id` — soft-delete.
- `POST /api/v1/filters/:filter_id/restore` — restore.
- `DELETE /api/v1/filters/:filter_id/hard` — hard-delete, если нет references.

Create request:

```json
{
  "identity": "flight-filter",
  "displayName": "Flight Filter",
  "folderIdentity": "root-filters",
  "fields": [
    {
      "key": "status",
      "mode": "static",
      "staticOptions": [{ "value": "active", "label": "Active" }],
      "multiple": true,
      "defaultValue": "*",
      "active": true
    }
  ],
  "source": "defineFilter({})",
  "sourceVersion": 1,
  "meta": {},
  "active": true
}
```

Arrays in PATCH replace completely. Backend validates structural fields and references but does not compile source.

Errors: `filter_not_found`, `filter_identity_conflict`, `filter_in_use`, `invalid_filter_field`, `vocab_not_found`, `converter_not_found`, folder errors.

## Acceptance Criteria

Migration includes missing source/sourceVersion/workspace fields; both contracts round-trip independently; JSON identities индексируются без потери portability; hard-delete dependencies защищён; lifecycle/OpenAPI/tests готовы; `go test ./...` проходит.
