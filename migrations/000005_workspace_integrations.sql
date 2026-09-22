-- +goose Up
CREATE TABLE workspace_integrations
(
    workspace_id       UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    integration_id     UUID        NOT NULL REFERENCES integrations (id),
    version            TEXT        NOT NULL,
    configuration      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID        NOT NULL REFERENCES service_users (id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, integration_id)
);

-- +goose Down
DROP TABLE IF EXISTS workspace_integrations;
