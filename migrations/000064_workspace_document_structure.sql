-- +goose Up
ALTER TABLE workspaces
    ADD COLUMN document_structure TEXT NOT NULL DEFAULT 'frontend'
        CHECK (document_structure IN ('frontend', 'custom'));

-- +goose Down
ALTER TABLE workspaces
    DROP COLUMN IF EXISTS document_structure;
