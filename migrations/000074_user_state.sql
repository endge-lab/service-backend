-- +goose Up
CREATE SEQUENCE user_state_revision AS bigint;

CREATE TABLE user_state (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    environment text NOT NULL,
    facet_scope text NOT NULL,
    user_id uuid NOT NULL REFERENCES service_users(id) ON DELETE CASCADE,
    key text NOT NULL,
    value jsonb,
    revision bigint NOT NULL DEFAULT nextval('user_state_revision'),
    mutation_id text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid REFERENCES service_users(id) ON DELETE SET NULL,
    PRIMARY KEY (workspace_id, environment, facet_scope, user_id, key)
);

CREATE TABLE user_state_resets (
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    environment text NOT NULL,
    facet_scope text NOT NULL,
    user_id text NOT NULL,
    revision bigint NOT NULL DEFAULT nextval('user_state_revision'),
    PRIMARY KEY (workspace_id, environment, facet_scope, user_id)
);

CREATE TABLE user_state_admin_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    environment text NOT NULL,
    facet_scope text NOT NULL,
    actor_id uuid REFERENCES service_users(id) ON DELETE SET NULL,
    target_user_id uuid REFERENCES service_users(id) ON DELETE SET NULL,
    action text NOT NULL,
    affected_keys text[] NOT NULL,
    affected_count bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_state_admin_events, user_state_resets, user_state;
DROP SEQUENCE user_state_revision;
