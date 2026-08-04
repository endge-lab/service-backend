-- +goose Up
INSERT INTO projects (
    id, workspace_id, identity, display_name, folder_id, data, managed_by, created_by, updated_by
)
SELECT '00000000-0000-0000-0000-000000000100',
       '00000000-0000-0000-0000-000000000010',
       'default',
       'Default Project',
       f.id,
       '{"configuration":{"mode":"inherit","patch":{}}}'::jsonb,
       'system',
       '00000000-0000-0000-0000-000000000001',
       '00000000-0000-0000-0000-000000000001'
FROM folders f
WHERE f.workspace_id = '00000000-0000-0000-0000-000000000010'
  AND f.entity_type = 'projects'
  AND f.is_root
ON CONFLICT (workspace_id, identity) DO NOTHING;

INSERT INTO tenants (
    id, workspace_id, identity, display_name, folder_id, data, managed_by, created_by, updated_by, code
)
SELECT '00000000-0000-0000-0000-000000000101',
       '00000000-0000-0000-0000-000000000010',
       'default',
       'Default Tenant',
       f.id,
       '{"configuration":{"mode":"inherit","patch":{}}}'::jsonb,
       'system',
       '00000000-0000-0000-0000-000000000001',
       '00000000-0000-0000-0000-000000000001',
       'default'
FROM folders f
WHERE f.workspace_id = '00000000-0000-0000-0000-000000000010'
  AND f.entity_type = 'tenants'
  AND f.is_root
ON CONFLICT (workspace_id, identity) DO NOTHING;

INSERT INTO environments (
    id, workspace_id, identity, display_name, folder_id, data, managed_by, created_by, updated_by
)
SELECT '00000000-0000-0000-0000-000000000102',
       '00000000-0000-0000-0000-000000000010',
       'dev',
       'Development',
       f.id,
       '{"configuration":{"mode":"inherit","patch":{}}}'::jsonb,
       'system',
       '00000000-0000-0000-0000-000000000001',
       '00000000-0000-0000-0000-000000000001'
FROM folders f
WHERE f.workspace_id = '00000000-0000-0000-0000-000000000010'
  AND f.entity_type = 'environments'
  AND f.is_root
ON CONFLICT (workspace_id, identity) DO NOTHING;

INSERT INTO project_environments (workspace_id, project_id, environment_id, sort_order)
SELECT '00000000-0000-0000-0000-000000000010', p.id, e.id, 0
FROM projects p
JOIN environments e ON e.workspace_id = p.workspace_id
WHERE p.workspace_id = '00000000-0000-0000-0000-000000000010'
  AND p.identity = 'default'
  AND e.identity = 'dev'
ON CONFLICT (workspace_id, project_id, environment_id) DO NOTHING;

-- +goose Down
DELETE FROM project_environments
WHERE workspace_id = '00000000-0000-0000-0000-000000000010'
  AND project_id = '00000000-0000-0000-0000-000000000100'
  AND environment_id = '00000000-0000-0000-0000-000000000102';

DELETE FROM environments
WHERE id = '00000000-0000-0000-0000-000000000102'
  AND workspace_id = '00000000-0000-0000-0000-000000000010';

DELETE FROM tenants
WHERE id = '00000000-0000-0000-0000-000000000101'
  AND workspace_id = '00000000-0000-0000-0000-000000000010';

DELETE FROM projects
WHERE id = '00000000-0000-0000-0000-000000000100'
  AND workspace_id = '00000000-0000-0000-0000-000000000010';
