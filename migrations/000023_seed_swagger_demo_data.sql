-- +goose Up
-- Stable demo graph for exercising the OpenAPI examples locally.
INSERT INTO workspaces (id, identity, display_name, configuration)
VALUES (
    '10000000-0000-4000-8000-000000000001',
    'demo-workspace',
    'Demo Workspace',
    '{
      "vars": [{"name": "API_URL", "defaultValue": "https://api.example.test"}],
      "locales": [
        {"code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr"},
        {"code": "en", "displayName": "English", "shortLabel": "EN", "direction": "ltr"}
      ],
      "defaultLocale": "ru",
      "fallbackLocale": "ru",
      "themes": [
        {"identity": "light", "displayName": "Светлая"},
        {"identity": "dark", "displayName": "Тёмная"}
      ],
      "defaultTheme": "light",
      "defaultAuthProfileIdentity": null,
      "sfcAdapterIds": ["native-vue"],
      "defaultSfcAdapterId": "native-vue"
    }'::jsonb
);

INSERT INTO projects (id, workspace_id, identity, display_name, description, active, deleted_at, meta)
VALUES
    ('10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000001', 'demo-project', 'Demo Project', 'Project used by Swagger examples', TRUE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000003', '10000000-0000-4000-8000-000000000001', 'restore-project', 'Restore Project', 'Use this project to exercise restore', FALSE, NOW(), '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000004', '10000000-0000-4000-8000-000000000001', 'hard-delete-project', 'Hard delete project', 'Use this project to exercise hard delete', FALSE, NOW(), '{"seed": true}'::jsonb);

INSERT INTO folders (id, workspace_id, project_id, entity_type, identity, display_name, description, parent_id, is_root, is_system, deleted_at, meta)
VALUES
    ('10000000-0000-4000-8000-000000000011', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'components-legacy', 'root-components-legacy', 'root-components-legacy', NULL, NULL, TRUE, TRUE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000012', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'converters', 'root-converters', 'root-converters', NULL, NULL, TRUE, TRUE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000013', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'queries', 'root-queries', 'root-queries', NULL, NULL, TRUE, TRUE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000014', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'data-views', 'root-data-views', 'root-data-views', NULL, NULL, TRUE, TRUE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000015', '10000000-0000-4000-8000-000000000001', NULL, 'tenants', 'root-tenants', 'root-tenants', NULL, NULL, TRUE, TRUE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000021', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'components-legacy', 'shared-components-legacy', 'Shared legacy components', 'Folder used by the component examples', '10000000-0000-4000-8000-000000000011', FALSE, FALSE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000022', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'converters', 'shared-converters', 'Shared converters', 'Folder used by the converter examples', '10000000-0000-4000-8000-000000000012', FALSE, FALSE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000023', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'queries', 'shared-queries', 'Shared queries', 'Folder used by the query examples', '10000000-0000-4000-8000-000000000013', FALSE, FALSE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000024', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'data-views', 'shared-data-views', 'Shared data views', 'Folder used by the data-view examples', '10000000-0000-4000-8000-000000000014', FALSE, FALSE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000025', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'components-legacy', 'restore-components-legacy', 'Restore components', 'Use this folder to exercise restore', '10000000-0000-4000-8000-000000000011', FALSE, FALSE, NOW(), '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000026', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', 'components-legacy', 'hard-delete-components-legacy', 'Hard delete components', 'Use this folder to exercise hard delete', '10000000-0000-4000-8000-000000000011', FALSE, FALSE, NOW(), '{"seed": true}'::jsonb);

INSERT INTO components_legacy (id, workspace_id, project_id, folder_id, identity, display_name, description, component_type, source, props_schema, bindings, meta, active, deleted_at)
VALUES
    ('10000000-0000-4000-8000-000000000031', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000021', 'user-card', 'User Card', 'SFC component for a user card', 'component-sfc', '<template><article>{{ user.name }}</article></template>', '{"name": {"type": "string", "required": true}}'::jsonb, '{}'::jsonb, '{"seed": true}'::jsonb, TRUE, NULL),
    ('10000000-0000-4000-8000-000000000032', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000021', 'restore-user-card', 'Restore User Card', 'Use this component to exercise restore', 'component-sfc', '<template><article>Restore</article></template>', '{}'::jsonb, '{}'::jsonb, '{"seed": true}'::jsonb, FALSE, NOW()),
    ('10000000-0000-4000-8000-000000000033', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000021', 'hard-delete-user-card', 'Hard delete User Card', 'Use this component to exercise hard delete', 'component-sfc', '<template><article>Hard delete</article></template>', '{}'::jsonb, '{}'::jsonb, '{"seed": true}'::jsonb, FALSE, NOW());

INSERT INTO converters (id, workspace_id, project_id, folder_id, identity, display_name, description, converter_type, source, is_system, meta, active, deleted_at)
VALUES
    ('10000000-0000-4000-8000-000000000041', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000022', 'date-to-string', 'Date to string', 'Formats an ISO date', 'javascript', '{"expression": "new Date(value).toISOString()"}'::jsonb, FALSE, '{"seed": true}'::jsonb, TRUE, NULL),
    ('10000000-0000-4000-8000-000000000042', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000022', 'restore-date-to-string', 'Restore date converter', 'Use this converter to exercise restore', 'javascript', '{"expression": "value"}'::jsonb, FALSE, '{"seed": true}'::jsonb, FALSE, NOW()),
    ('10000000-0000-4000-8000-000000000043', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000022', 'hard-delete-date-to-string', 'Hard delete date converter', 'Use this converter to exercise hard delete', 'javascript', '{"expression": "value"}'::jsonb, FALSE, '{"seed": true}'::jsonb, FALSE, NOW());

INSERT INTO queries (id, workspace_id, project_id, folder_id, identity, display_name, description, query_type, source, params, headers, auth, timeout_ms, mock_data, mock_data_enabled, active, deleted_at, meta)
VALUES
    ('10000000-0000-4000-8000-000000000051', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000023', 'users-list', 'Users list', 'Loads users for the table', 'http', '{"method": "GET", "url": "https://example.test/users"}'::jsonb, '[]'::jsonb, '{}'::jsonb, NULL, 5000, '{"items": []}'::jsonb, TRUE, TRUE, NULL, '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000052', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000023', 'restore-users-list', 'Restore users list', 'Use this query to exercise restore', 'http', '{"method": "GET", "url": "https://example.test/restore-users"}'::jsonb, '[]'::jsonb, '{}'::jsonb, NULL, 5000, NULL, FALSE, FALSE, NOW(), '{"seed": true}'::jsonb),
    ('10000000-0000-4000-8000-000000000053', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000023', 'hard-delete-users-list', 'Hard delete users list', 'Use this query to exercise hard delete', 'http', '{"method": "GET", "url": "https://example.test/hard-delete-users"}'::jsonb, '[]'::jsonb, '{}'::jsonb, NULL, 5000, NULL, FALSE, FALSE, NOW(), '{"seed": true}'::jsonb);

INSERT INTO data_views (id, workspace_id, project_id, folder_id, query_id, identity, display_name, description, view_type, source, input_schema, output_schema, meta, active, deleted_at)
VALUES
    ('10000000-0000-4000-8000-000000000061', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000024', '10000000-0000-4000-8000-000000000051', 'users-table', 'Users table', 'Table view for users', 'table', '{"columns": ["name", "email"]}'::jsonb, '{"type": "object"}'::jsonb, '{"type": "object"}'::jsonb, '{"seed": true}'::jsonb, TRUE, NULL),
    ('10000000-0000-4000-8000-000000000062', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000024', '10000000-0000-4000-8000-000000000051', 'restore-users-table', 'Restore users table', 'Use this data view to exercise restore', 'table', '{"columns": ["name"]}'::jsonb, '{"type": "object"}'::jsonb, '{"type": "object"}'::jsonb, '{"seed": true}'::jsonb, FALSE, NOW()),
    ('10000000-0000-4000-8000-000000000063', '10000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000024', '10000000-0000-4000-8000-000000000051', 'hard-delete-users-table', 'Hard delete users table', 'Use this data view to exercise hard delete', 'table', '{"columns": ["name"]}'::jsonb, '{"type": "object"}'::jsonb, '{"type": "object"}'::jsonb, '{"seed": true}'::jsonb, FALSE, NOW());

-- +goose Down
DELETE FROM data_views WHERE workspace_id = '10000000-0000-4000-8000-000000000001';
DELETE FROM components_legacy WHERE workspace_id = '10000000-0000-4000-8000-000000000001';
DELETE FROM converters WHERE workspace_id = '10000000-0000-4000-8000-000000000001';
DELETE FROM queries WHERE workspace_id = '10000000-0000-4000-8000-000000000001';
DELETE FROM folders WHERE workspace_id = '10000000-0000-4000-8000-000000000001';
DELETE FROM projects WHERE workspace_id = '10000000-0000-4000-8000-000000000001';
DELETE FROM workspaces WHERE id = '10000000-0000-4000-8000-000000000001';
