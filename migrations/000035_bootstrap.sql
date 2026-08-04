-- +goose Up
INSERT INTO service_users (id, provider_id, subject, issuer, username, display_name, active, is_system)
VALUES ('00000000-0000-0000-0000-000000000001', 'system', 'system', 'urn:endge:system', 'system', 'Endge System', TRUE,
        TRUE);

INSERT INTO workspaces (id, identity, display_name, description, data_mode, created_by, updated_by)
VALUES ('00000000-0000-0000-0000-000000000010', 'default', 'Default', 'Default Endge workspace', 'development',
        '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001');

INSERT INTO folders (workspace_id, identity, display_name, entity_type, is_root, managed_by, created_by, updated_by)
SELECT '00000000-0000-0000-0000-000000000010',
       'root-' || entity_type,
       'Root ' || entity_type,
       entity_type,
       TRUE,
       'system',
       '00000000-0000-0000-0000-000000000001',
       '00000000-0000-0000-0000-000000000001'
FROM unnest(ARRAY ['projects','tenants','environments','types','queries','data-views','compositions','stores','streams','updates','mocks','components','actions','filters','converters','computations','vocabs','i18n-bundles','auth-profiles','navigations','styles']) entity_type;

INSERT INTO workspace_commits (workspace_id, base_sequence, head_sequence, message, revision_policy, operation,
                               created_by)
VALUES ('00000000-0000-0000-0000-000000000010', 0, 0, 'Initial workspace state', 'preserve', 'bootstrap',
        '00000000-0000-0000-0000-000000000001');

-- +goose Down
DELETE
FROM releases
WHERE workspace_id = '00000000-0000-0000-0000-000000000010';
UPDATE document_revisions
SET committed_in_commit_id = NULL
WHERE workspace_id = '00000000-0000-0000-0000-000000000010';
DELETE
FROM workspace_commits
WHERE workspace_id = '00000000-0000-0000-0000-000000000010';
DELETE
FROM document_revisions
WHERE workspace_id = '00000000-0000-0000-0000-000000000010';
DELETE
FROM mutation_batches
WHERE workspace_id = '00000000-0000-0000-0000-000000000010';
-- +goose StatementBegin
DO
$$
    DECLARE
        table_name TEXT;
    BEGIN
        FOREACH table_name IN ARRAY ARRAY ['project_environments']
            LOOP
                IF to_regclass(table_name) IS NOT NULL THEN
                    EXECUTE format('DELETE FROM %I WHERE workspace_id = $1', table_name)
                        USING '00000000-0000-0000-0000-000000000010'::uuid;
                END IF;
            END LOOP;
        FOREACH table_name IN ARRAY ARRAY ['projects','tenants','environments','types','queries','data_views','compositions','stores','streams','updates','mocks','components','actions','filters','converters','computations','vocabs','i18n_bundles','auth_profiles','navigations','styles']
            LOOP
                EXECUTE format('DELETE FROM %I WHERE workspace_id = $1', table_name)
                    USING '00000000-0000-0000-0000-000000000010'::uuid;
            END LOOP;
    END
$$;
-- +goose StatementEnd
DELETE
FROM folders
WHERE workspace_id = '00000000-0000-0000-0000-000000000010';
DELETE
FROM workspaces
WHERE id = '00000000-0000-0000-0000-000000000010';
DELETE
FROM service_users u
WHERE u.id = '00000000-0000-0000-0000-000000000001'
  AND NOT EXISTS (SELECT 1 FROM integrations i WHERE i.created_by = u.id OR i.updated_by = u.id)
  AND NOT EXISTS (SELECT 1 FROM document_revisions r WHERE r.created_by = u.id)
  AND NOT EXISTS (SELECT 1 FROM mutation_batches b WHERE b.actor_user_id = u.id);
