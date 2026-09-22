-- +goose Up
CREATE TABLE simulations
(
    id                 UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id       UUID        NOT NULL,
    identity           TEXT        NOT NULL,
    display_name       TEXT        NOT NULL,
    description        TEXT,
    folder_id          UUID,
    data               JSONB       NOT NULL DEFAULT '{}'::jsonb,
    managed_by         TEXT        NOT NULL DEFAULT 'user' CHECK (managed_by IN ('user', 'system', 'integration')),
    managed_by_id      TEXT,
    meta               JSONB       NOT NULL DEFAULT '{}'::jsonb,
    active             BOOLEAN     NOT NULL DEFAULT TRUE,
    deleted_at         TIMESTAMPTZ,
    created_by UUID        NOT NULL,
    updated_by UUID        NOT NULL,
    revision           INTEGER     NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, identity),
    UNIQUE (workspace_id, id),
    CONSTRAINT simulations_workspace_fk FOREIGN KEY (workspace_id) REFERENCES workspaces (id),
    CONSTRAINT simulations_created_by_fk FOREIGN KEY (created_by) REFERENCES service_users (id),
    CONSTRAINT simulations_updated_by_fk FOREIGN KEY (updated_by) REFERENCES service_users (id),
    CONSTRAINT simulations_folder_fk FOREIGN KEY (workspace_id, folder_id) REFERENCES folders (workspace_id, id)
);

INSERT INTO folders (workspace_id, identity, display_name, entity_type, is_root, managed_by, created_by, updated_by)
SELECT w.id, 'root-simulations', 'Root simulations', 'simulations', TRUE, 'system', w.created_by, w.created_by
FROM workspaces w
ON CONFLICT (workspace_id, identity) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS simulations;
DELETE FROM folders WHERE identity = 'root-simulations' AND entity_type = 'simulations' AND is_root AND managed_by = 'system';
