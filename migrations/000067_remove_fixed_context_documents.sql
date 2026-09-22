-- +goose Up
WITH required(identity, display_name, icon, color, ordinal) AS (
    VALUES
        ('tenant', 'Tenant', 'Building2', '#7c3aed', 1),
        ('project', 'Project', 'FolderKanban', '#2563eb', 2),
        ('environment', 'Environment', 'Layers3', '#16a34a', 3)
),
revived AS (
    SELECT
        facet.id,
        COALESCE((
            SELECT max(active_facet.position)
            FROM facets active_facet
            WHERE active_facet.workspace_id = facet.workspace_id
              AND active_facet.deleted_at IS NULL
        ), -1) + row_number() OVER (
            PARTITION BY facet.workspace_id
            ORDER BY required.ordinal
        ) AS position
    FROM facets facet
    JOIN required ON required.identity = facet.identity
    WHERE facet.deleted_at IS NOT NULL
)
UPDATE facets facet
SET position = revived.position,
    active = TRUE,
    deleted_at = NULL,
    updated_at = NOW(),
    revision = facet.revision + 1
FROM revived
WHERE revived.id = facet.id;

WITH required(identity, display_name, icon, color, ordinal) AS (
    VALUES
        ('tenant', 'Tenant', 'Building2', '#7c3aed', 1),
        ('project', 'Project', 'FolderKanban', '#2563eb', 2),
        ('environment', 'Environment', 'Layers3', '#16a34a', 3)
),
missing AS (
    SELECT
        workspace.id AS workspace_id,
        required.identity,
        required.display_name,
        required.icon,
        required.color,
        COALESCE((
            SELECT max(active_facet.position)
            FROM facets active_facet
            WHERE active_facet.workspace_id = workspace.id
              AND active_facet.deleted_at IS NULL
        ), -1) + row_number() OVER (
            PARTITION BY workspace.id
            ORDER BY required.ordinal
        ) AS position,
        workspace.created_by,
        workspace.updated_by
    FROM workspaces workspace
    CROSS JOIN required
    WHERE NOT EXISTS (
        SELECT 1
        FROM facets existing
        WHERE existing.workspace_id = workspace.id
          AND existing.identity = required.identity
    )
)
INSERT INTO facets (
    workspace_id,
    identity,
    display_name,
    icon,
    color,
    position,
    created_by,
    updated_by
)
SELECT
    workspace_id,
    identity,
    display_name,
    icon,
    color,
    position,
    created_by,
    updated_by
FROM missing;

WITH legacy_documents AS (
    SELECT
        source.workspace_id,
        'tenant'::text AS facet_identity,
        source.identity,
        source.display_name,
        source.description,
        COALESCE(source.data -> 'configuration', '{"mode":"inherit","patch":{}}'::jsonb) AS configuration,
        source.meta,
        source.active,
        source.deleted_at,
        source.created_by,
        source.updated_by,
        source.revision,
        source.created_at,
        source.updated_at
    FROM tenants source
    UNION ALL
    SELECT
        source.workspace_id,
        'project'::text,
        source.identity,
        source.display_name,
        source.description,
        COALESCE(source.data -> 'configuration', '{"mode":"inherit","patch":{}}'::jsonb),
        source.meta,
        source.active,
        source.deleted_at,
        source.created_by,
        source.updated_by,
        source.revision,
        source.created_at,
        source.updated_at
    FROM projects source
    UNION ALL
    SELECT
        source.workspace_id,
        'environment'::text,
        source.identity,
        source.display_name,
        source.description,
        COALESCE(source.data -> 'configuration', '{"mode":"inherit","patch":{}}'::jsonb),
        source.meta,
        source.active,
        source.deleted_at,
        source.created_by,
        source.updated_by,
        source.revision,
        source.created_at,
        source.updated_at
    FROM environments source
)
INSERT INTO facet_documents AS target (
    workspace_id,
    facet_id,
    identity,
    display_name,
    description,
    configuration,
    meta,
    active,
    deleted_at,
    created_by,
    updated_by,
    revision,
    created_at,
    updated_at
)
SELECT
    source.workspace_id,
    facet.id,
    source.identity,
    source.display_name,
    source.description,
    source.configuration,
    source.meta,
    source.active,
    source.deleted_at,
    source.created_by,
    source.updated_by,
    source.revision,
    source.created_at,
    source.updated_at
FROM legacy_documents source
JOIN facets facet
  ON facet.workspace_id = source.workspace_id
 AND facet.identity = source.facet_identity
 AND facet.deleted_at IS NULL
ON CONFLICT (facet_id, identity) DO UPDATE
SET display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    configuration = EXCLUDED.configuration,
    meta = EXCLUDED.meta,
    active = EXCLUDED.active,
    deleted_at = EXCLUDED.deleted_at,
    updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at,
    revision = target.revision + 1;

INSERT INTO compositions (
    workspace_id,
    identity,
    display_name,
    description,
    folder_id,
    workspace_folder_id,
    data,
    managed_by,
    managed_by_id,
    meta,
    active,
    deleted_at,
    created_by,
    updated_by,
    revision,
    created_at,
    updated_at
)
SELECT
    source.workspace_id,
    source.identity,
    source.display_name,
    source.description,
    NULL,
    source.workspace_folder_id,
    source.data - 'configuration' - 'slug' - 'order',
    source.managed_by,
    source.managed_by_id,
    source.meta,
    source.active,
    source.deleted_at,
    source.created_by,
    source.updated_by,
    source.revision,
    source.created_at,
    source.updated_at
FROM projects source
ON CONFLICT (workspace_id, identity) DO NOTHING;

-- The globally bootstrapped Project/Environment pair carries no authored
-- compatibility information and can be removed without a manual data step.
DELETE FROM project_environments
WHERE workspace_id = '00000000-0000-0000-0000-000000000010'
  AND project_id = '00000000-0000-0000-0000-000000000100'
  AND environment_id = '00000000-0000-0000-0000-000000000102';

INSERT INTO compositions (
    workspace_id,
    identity,
    display_name,
    folder_id,
    workspace_folder_id,
    data,
    managed_by,
    created_by,
    updated_by
)
SELECT
    w.id,
    CASE
        WHEN occupied.id IS NULL THEN 'workspace-startup'
        ELSE 'workspace-startup-migrated-' || replace(w.id::text, '-', '')
    END,
    'Стартовая композиция',
    collection_root.id,
    workspace_root.id,
    jsonb_build_object(
        'source', E'defineComposition({\n  activateOn: startup(),\n  data: {},\n  resources: {},\n  runtimes: {},\n  hooks: [],\n  outputs: {},\n})\n',
        'sourceVersion', 1
    ),
    'user',
    w.created_by,
    w.updated_by
FROM workspaces w
JOIN folders collection_root
  ON collection_root.workspace_id = w.id
 AND collection_root.identity = 'root-compositions'
 AND collection_root.scope = 'collection'
 AND collection_root.deleted_at IS NULL
JOIN folders workspace_root
  ON workspace_root.workspace_id = w.id
 AND workspace_root.identity = 'root-workspace-files'
 AND workspace_root.scope = 'workspace'
 AND workspace_root.deleted_at IS NULL
LEFT JOIN compositions occupied
  ON occupied.workspace_id = w.id
 AND occupied.identity = 'workspace-startup'
WHERE w.active
  AND NOT EXISTS (
      SELECT 1
      FROM compositions candidate
      WHERE candidate.workspace_id = w.id
        AND candidate.deleted_at IS NULL
        AND candidate.active
  );

WITH candidates AS (
    SELECT DISTINCT ON (workspace_id)
        workspace_id,
        id
    FROM compositions
    WHERE deleted_at IS NULL
      AND active
    ORDER BY
        workspace_id,
        (identity = 'workspace-startup') DESC,
        created_at,
        id
)
UPDATE workspaces w
SET startup_composition_id = candidate.id
FROM candidates candidate
WHERE candidate.workspace_id = w.id
  AND w.active
  AND NOT EXISTS (
      SELECT 1
      FROM compositions current_startup
      WHERE current_startup.workspace_id = w.id
        AND current_startup.id = w.startup_composition_id
        AND current_startup.deleted_at IS NULL
        AND current_startup.active
  );

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM workspaces w
        LEFT JOIN compositions c
          ON c.workspace_id = w.id
         AND c.id = w.startup_composition_id
         AND c.deleted_at IS NULL
         AND c.active
        WHERE w.active
          AND (w.startup_composition_id IS NULL OR c.id IS NULL)
    ) THEN
        RAISE EXCEPTION 'fixed context removal requires an active startup Composition in every active Workspace';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM tenants source
        WHERE source.deleted_at IS NULL
          AND (
            source.code <> source.identity
            OR NOT EXISTS (
                SELECT 1
                FROM facets facet
                JOIN facet_documents target
                  ON target.workspace_id = facet.workspace_id
                 AND target.facet_id = facet.id
                 AND target.identity = source.identity
                 AND target.deleted_at IS NULL
                WHERE facet.workspace_id = source.workspace_id
                  AND facet.identity = 'tenant'
                  AND facet.deleted_at IS NULL
                  AND target.display_name = source.display_name
                  AND target.description IS NOT DISTINCT FROM source.description
                  AND target.configuration = COALESCE(
                      source.data -> 'configuration',
                      '{"mode":"inherit","patch":{}}'::jsonb
                  )
                  AND target.meta = source.meta
                  AND target.active = source.active
            )
          )
    ) THEN
        RAISE EXCEPTION 'unmigrated Tenant documents remain';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM projects source
        WHERE source.deleted_at IS NULL
          AND (
            EXISTS (
                SELECT 1
                FROM project_environments relation
                WHERE relation.workspace_id = source.workspace_id
                  AND relation.project_id = source.id
            )
            OR NULLIF(btrim(source.data ->> 'slug'), '') IS NOT NULL
            OR COALESCE((source.data ->> 'order')::integer, 0) <> 0
            OR NOT EXISTS (
                SELECT 1
                FROM facets facet
                JOIN facet_documents target
                  ON target.workspace_id = facet.workspace_id
                 AND target.facet_id = facet.id
                 AND target.identity = source.identity
                 AND target.deleted_at IS NULL
                WHERE facet.workspace_id = source.workspace_id
                  AND facet.identity = 'project'
                  AND facet.deleted_at IS NULL
                  AND target.display_name = source.display_name
                  AND target.description IS NOT DISTINCT FROM source.description
                  AND target.configuration = COALESCE(
                      source.data -> 'configuration',
                      '{"mode":"inherit","patch":{}}'::jsonb
                  )
                  AND target.meta = source.meta
                  AND target.active = source.active
            )
            OR NOT EXISTS (
                SELECT 1
                FROM compositions target
                WHERE target.workspace_id = source.workspace_id
                  AND target.identity = source.identity
                  AND target.deleted_at IS NULL
                  AND target.display_name = source.display_name
                  AND target.description IS NOT DISTINCT FROM source.description
                  AND target.data ->> 'source' = source.data ->> 'source'
                  AND target.data ->> 'sourceVersion' = source.data ->> 'sourceVersion'
                  AND target.meta = source.meta
                  AND target.active = source.active
            )
          )
    ) THEN
        RAISE EXCEPTION 'unmigrated Project documents remain';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM environments source
        WHERE source.deleted_at IS NULL
          AND NOT EXISTS (
              SELECT 1
              FROM facets facet
              JOIN facet_documents target
                ON target.workspace_id = facet.workspace_id
               AND target.facet_id = facet.id
               AND target.identity = source.identity
               AND target.deleted_at IS NULL
              WHERE facet.workspace_id = source.workspace_id
                AND facet.identity = 'environment'
                AND facet.deleted_at IS NULL
                AND target.display_name = source.display_name
                AND target.description IS NOT DISTINCT FROM source.description
                AND target.configuration = COALESCE(
                    source.data -> 'configuration',
                    '{"mode":"inherit","patch":{}}'::jsonb
                )
                AND target.meta = source.meta
                AND target.active = source.active
          )
    ) THEN
        RAISE EXCEPTION 'unmigrated Environment documents remain';
    END IF;

    UPDATE simulations
    SET data = jsonb_set(
            data,
            '{source}',
            to_jsonb(regexp_replace(
                data ->> 'source',
                '(^|[^A-Za-z0-9_$])project([[:space:]]*\()',
                '\1composition\2',
                'g'
            )),
            false
        ),
        revision = revision + 1,
        updated_at = NOW()
    WHERE data ->> 'source' ~ '(^|[^A-Za-z0-9_$])project[[:space:]]*\(';

    IF EXISTS (
        SELECT 1
        FROM simulations
        WHERE data ->> 'source' ~ '(^|[^A-Za-z0-9_$])project[[:space:]]*\('
    ) THEN
        RAISE EXCEPTION 'Simulation Source still contains project(...) references';
    END IF;
END $$;
-- +goose StatementEnd

DROP TABLE project_environments;
DROP TABLE projects;
DROP TABLE tenants;
DROP TABLE environments;

DELETE FROM folders
WHERE scope = 'collection'
  AND entity_type IN ('projects', 'tenants', 'environments');

-- +goose Down
-- The tables contained authored data and cannot be reconstructed after removal.
-- +goose StatementBegin
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000067 is irreversible; restore the pre-migration database backup';
END $$;
-- +goose StatementEnd
