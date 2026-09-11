-- +goose Up
ALTER TABLE folders
    ADD COLUMN scope TEXT NOT NULL DEFAULT 'collection'
        CHECK (scope IN ('collection', 'workspace')),
    ADD COLUMN icon TEXT
        CHECK (icon IS NULL OR icon ~ '^[A-Z][A-Za-z0-9]*$'),
    ADD COLUMN color TEXT
        CHECK (color IS NULL OR color ~ '^#[0-9a-f]{6}$'),
    ALTER COLUMN entity_type DROP NOT NULL;

ALTER TABLE folders
    ADD CONSTRAINT folders_scope_entity_type_check CHECK (
        (scope = 'collection' AND entity_type IS NOT NULL AND btrim(entity_type) <> '')
        OR (scope = 'workspace' AND entity_type IS NULL)
    ),
    ADD CONSTRAINT folders_workspace_parent_scope_key UNIQUE (workspace_id, id, scope),
    ADD CONSTRAINT folders_workspace_parent_scope_fk
        FOREIGN KEY (workspace_id, parent_id, scope)
        REFERENCES folders (workspace_id, id, scope)
        DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM folders
        WHERE identity = 'root-workspace-files'
          AND NOT (
              scope = 'workspace'
              AND entity_type IS NULL
              AND parent_id IS NULL
              AND is_root
              AND managed_by = 'system'
              AND deleted_at IS NULL
          )
    ) THEN
        RAISE EXCEPTION 'reserved folder identity root-workspace-files is already used by an incompatible folder';
    END IF;
END $$;
-- +goose StatementEnd

INSERT INTO folders (
    workspace_id,
    identity,
    display_name,
    entity_type,
    scope,
    parent_id,
    is_root,
    managed_by,
    icon,
    color,
    created_by,
    updated_by
)
SELECT
    w.id,
    'root-workspace-files',
    'Workspace',
    NULL,
    'workspace',
    NULL,
    TRUE,
    'system',
    NULL,
    NULL,
    w.created_by,
    w.updated_by
FROM workspaces w
WHERE NOT EXISTS (
    SELECT 1
    FROM folders f
    WHERE f.workspace_id = w.id
      AND f.identity = 'root-workspace-files'
);

-- Keep the schema-visible ALTER statements explicit: sqlc cannot infer columns
-- added through dynamic SQL inside a DO block.
ALTER TABLE projects ADD COLUMN workspace_folder_id UUID;
ALTER TABLE tenants ADD COLUMN workspace_folder_id UUID;
ALTER TABLE environments ADD COLUMN workspace_folder_id UUID;
ALTER TABLE types ADD COLUMN workspace_folder_id UUID;
ALTER TABLE queries ADD COLUMN workspace_folder_id UUID;
ALTER TABLE data_views ADD COLUMN workspace_folder_id UUID;
ALTER TABLE compositions ADD COLUMN workspace_folder_id UUID;
ALTER TABLE stores ADD COLUMN workspace_folder_id UUID;
ALTER TABLE streams ADD COLUMN workspace_folder_id UUID;
ALTER TABLE simulations ADD COLUMN workspace_folder_id UUID;
ALTER TABLE updates ADD COLUMN workspace_folder_id UUID;
ALTER TABLE mocks ADD COLUMN workspace_folder_id UUID;
ALTER TABLE components ADD COLUMN workspace_folder_id UUID;
ALTER TABLE actions ADD COLUMN workspace_folder_id UUID;
ALTER TABLE filters ADD COLUMN workspace_folder_id UUID;
ALTER TABLE converters ADD COLUMN workspace_folder_id UUID;
ALTER TABLE computations ADD COLUMN workspace_folder_id UUID;
ALTER TABLE vocabs ADD COLUMN workspace_folder_id UUID;
ALTER TABLE i18n_bundles ADD COLUMN workspace_folder_id UUID;
ALTER TABLE auth_profiles ADD COLUMN workspace_folder_id UUID;
ALTER TABLE navigations ADD COLUMN workspace_folder_id UUID;
ALTER TABLE styles ADD COLUMN workspace_folder_id UUID;
ALTER TABLE configurations ADD COLUMN workspace_folder_id UUID;

-- +goose StatementBegin
DO $$
DECLARE
    table_name TEXT;
    constraint_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'projects', 'tenants', 'environments', 'types', 'queries',
        'data_views', 'compositions', 'stores', 'streams', 'simulations',
        'updates', 'mocks', 'components', 'actions', 'filters', 'converters',
        'computations', 'vocabs', 'i18n_bundles', 'auth_profiles',
        'navigations', 'styles', 'configurations'
    ]
    LOOP
        EXECUTE format(
            'UPDATE %I d SET workspace_folder_id = f.id FROM folders f WHERE f.workspace_id = d.workspace_id AND f.identity = %L AND f.scope = %L',
            table_name,
            'root-workspace-files',
            'workspace'
        );
        EXECUTE format('ALTER TABLE %I ALTER COLUMN workspace_folder_id SET NOT NULL', table_name);
        constraint_name := table_name || '_workspace_folder_fk';
        EXECUTE format(
            'ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (workspace_id, workspace_folder_id) REFERENCES folders (workspace_id, id)',
            table_name,
            constraint_name
        );
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'projects', 'tenants', 'environments', 'types', 'queries',
        'data_views', 'compositions', 'stores', 'streams', 'simulations',
        'updates', 'mocks', 'components', 'actions', 'filters', 'converters',
        'computations', 'vocabs', 'i18n_bundles', 'auth_profiles',
        'navigations', 'styles', 'configurations'
    ]
    LOOP
        EXECUTE format('ALTER TABLE %I DROP COLUMN IF EXISTS workspace_folder_id', table_name);
    END LOOP;
END $$;
-- +goose StatementEnd

DELETE FROM folders
WHERE scope = 'workspace';

ALTER TABLE folders
    DROP CONSTRAINT IF EXISTS folders_workspace_parent_scope_fk,
    DROP CONSTRAINT IF EXISTS folders_workspace_parent_scope_key,
    DROP CONSTRAINT IF EXISTS folders_scope_entity_type_check,
    DROP COLUMN IF EXISTS color,
    DROP COLUMN IF EXISTS icon,
    DROP COLUMN IF EXISTS scope,
    ALTER COLUMN entity_type SET NOT NULL;
