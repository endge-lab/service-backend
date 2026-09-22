-- +goose Up
ALTER TABLE projects ADD COLUMN navigation_id UUID;
ALTER TABLE projects
    ADD CONSTRAINT projects_navigation_fk
    FOREIGN KEY (workspace_id, navigation_id) REFERENCES navigations (workspace_id, id);

-- +goose Down
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_navigation_fk;
ALTER TABLE projects DROP COLUMN IF EXISTS navigation_id;
