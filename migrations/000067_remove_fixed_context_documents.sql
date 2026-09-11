-- +goose Up
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
