-- +goose Up
INSERT INTO folders (
    id,
    identity,
    display_name,
    entity_type,
    description,
    is_root,
    is_system,
    meta
)
VALUES (
    '00000000-0000-4000-8000-000000000002',
    'root-components',
    'Root Components',
    'components',
    'Swagger examples root folder',
    true,
    true,
    '{"source":"swagger-example"}'::jsonb
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO settings (
    id,
    identity,
    display_name,
    meta
)
VALUES (
    '00000000-0000-4000-8000-000000000003',
    'swagger-settings',
    'Swagger Settings',
    '{"source":"swagger-example"}'::jsonb
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO navigations (
    id,
    identity,
    display_name,
    folder_id,
    tree,
    meta
)
VALUES (
    '00000000-0000-4000-8000-000000000004',
    'swagger-navigation',
    'Swagger Navigation',
    '00000000-0000-4000-8000-000000000002',
    '[]'::jsonb,
    '{"source":"swagger-example"}'::jsonb
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO environments (
    id,
    identity,
    display_name,
    is_system,
    folder_id
)
VALUES (
    '00000000-0000-4000-8000-000000000005',
    'swagger-environment',
    'Swagger Environment',
    true,
    '00000000-0000-4000-8000-000000000002'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO projects (
    id,
    identity,
    display_name,
    description,
    active,
    meta
)
VALUES (
    '00000000-0000-4000-8000-000000000001',
    'swagger-project',
    'Swagger Project',
    'Project for Swagger examples',
    false,
    '{"source":"swagger-example"}'::jsonb
)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM projects
WHERE id = '00000000-0000-4000-8000-000000000001';

DELETE FROM environments
WHERE id = '00000000-0000-4000-8000-000000000005';

DELETE FROM navigations
WHERE id = '00000000-0000-4000-8000-000000000004';

DELETE FROM settings
WHERE id = '00000000-0000-4000-8000-000000000003';

ALTER TABLE folders DISABLE TRIGGER USER;

DELETE FROM folders
WHERE id = '00000000-0000-4000-8000-000000000002';

ALTER TABLE folders ENABLE TRIGGER USER;
