-- +goose Up
ALTER TABLE workspaces
    ADD COLUMN startup_composition_id UUID;

ALTER TABLE workspaces
    ADD CONSTRAINT workspaces_startup_composition_fk
        FOREIGN KEY (id, startup_composition_id)
        REFERENCES compositions (workspace_id, id);

-- +goose Down
ALTER TABLE workspaces
    DROP CONSTRAINT IF EXISTS workspaces_startup_composition_fk,
    DROP COLUMN IF EXISTS startup_composition_id;
