-- +goose Up
ALTER TABLE user_state DROP CONSTRAINT user_state_pkey;
ALTER TABLE user_state
    ALTER COLUMN workspace_id DROP NOT NULL,
    ADD COLUMN application text NOT NULL DEFAULT '',
    ADD CONSTRAINT user_state_scope CHECK (
        (workspace_id IS NOT NULL AND application = '')
        OR (workspace_id IS NULL AND application <> '' AND facet_scope = '{}')
    ),
    ADD CONSTRAINT user_state_key UNIQUE NULLS NOT DISTINCT
        (workspace_id, environment, facet_scope, user_id, key, application);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM user_state WHERE workspace_id IS NULL) THEN
        RAISE EXCEPTION 'Application settings must be preserved before rolling back this migration';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE user_state
    DROP CONSTRAINT user_state_key,
    DROP CONSTRAINT user_state_scope,
    DROP COLUMN application,
    ALTER COLUMN workspace_id SET NOT NULL,
    ADD PRIMARY KEY (workspace_id, environment, facet_scope, user_id, key);
