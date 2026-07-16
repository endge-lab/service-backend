# 19. Navigations Foundation

## Цель

Реализовать `RNavigation` — workspace-scoped navigation tree, optionally linked to project.

Задача зависит от `07-domain-relations-and-portable-import`.

## Поля

```text
id UUID, workspace_id UUID, project_id UUID NULL, folder_id UUID NULL,
identity TEXT, display_name TEXT, description TEXT NULL,
is_system BOOLEAN DEFAULT false, tree JSONB DEFAULT [], meta JSONB,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

Navigation node:

```json
{
  "id": "orders",
  "type": "link",
  "title": "Orders",
  "icon": "list",
  "hidden": false,
  "disabled": false,
  "collapsedTitle": null,
  "path": "/orders",
  "routeName": "orders",
  "external": false,
  "children": []
}
```

Allowed node types: `section | group | link`. `section/group` may contain children; `link` must not. Node IDs должны быть unique within tree. External link requires absolute URL in `path`; internal link may use `path` or `routeName`. Uniqueness `(workspace_id, identity)`.

Добавить folder type `navigations` и root `root-navigations`; validate project/folder workspace. System navigation immutable through public API.

## HTTP API

Все методы требуют `X-Endge-Workspace`.

- `GET /api/v1/navigations?project_identity=...&folder_identity=...` — summaries без tree.
- `POST /api/v1/navigations` — создать navigation.
- `GET /api/v1/navigations/:navigation_ref` — detail по UUID или identity с tree.
- `PATCH /api/v1/navigations/:navigation_id` — metadata update по UUID.
- `PUT /api/v1/navigations/:navigation_id/tree` — полностью заменить navigation tree.
- `DELETE /api/v1/navigations/:navigation_id` — hard-delete, если navigation не referenced.

Create request:

```json
{
  "identity": "main",
  "displayName": "Main Navigation",
  "description": "Application navigation",
  "projectIdentity": "demo-project",
  "folderIdentity": "root-navigations",
  "tree": [
    { "id": "orders", "type": "link", "title": "Orders", "path": "/orders", "external": false }
  ],
  "meta": {}
}
```

Tree request:

```json
{
  "tree": [
    {
      "id": "operations",
      "type": "section",
      "title": "Operations",
      "children": [
        { "id": "orders", "type": "link", "title": "Orders", "path": "/orders" }
      ]
    }
  ]
}
```

Errors: `navigation_not_found`, `navigation_identity_conflict`, `navigation_in_use`, `invalid_navigation_tree`, `system_navigation_mutation_forbidden`, project/folder errors.

## Acceptance Criteria

Tree recursively validated and replaced atomically; list excludes tree; project references protected; OpenAPI/tests готовы; `go test ./...` проходит.
