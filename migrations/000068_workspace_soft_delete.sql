-- +goose Up
ALTER TABLE workspaces
    ADD COLUMN deleted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE workspaces
    DROP COLUMN IF EXISTS deleted_at;
