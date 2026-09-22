-- +goose Up
CREATE TABLE folders
(
    id                 UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id       UUID        NOT NULL REFERENCES workspaces (id),
    identity           TEXT        NOT NULL,
    display_name       TEXT        NOT NULL,
    description        TEXT,
    entity_type        TEXT        NOT NULL,
    parent_id          UUID,
    is_root            BOOLEAN     NOT NULL DEFAULT FALSE,
    managed_by         TEXT        NOT NULL DEFAULT 'user' CHECK (managed_by IN ('user', 'system', 'integration')),
    managed_by_id      TEXT,
    meta               JSONB       NOT NULL DEFAULT '{}'::jsonb,
    active             BOOLEAN     NOT NULL DEFAULT TRUE,
    deleted_at         TIMESTAMPTZ,
    created_by UUID        NOT NULL REFERENCES service_users (id),
    updated_by UUID        NOT NULL REFERENCES service_users (id),
    revision           INTEGER     NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, identity),
    UNIQUE (workspace_id, id),
    FOREIGN KEY (workspace_id, parent_id) REFERENCES folders (workspace_id, id) DEFERRABLE INITIALLY DEFERRED
);

-- +goose Down
DROP TABLE IF EXISTS folders;
