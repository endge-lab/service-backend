-- +goose Up
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_navigation_fk;
ALTER TABLE projects DROP COLUMN IF EXISTS navigation_id;

-- Удаляем также старые JSON-представления связи, включая soft-deleted проекты.
UPDATE projects
SET data = data - 'navigation' - 'navigationId' - 'navigationIdentity'
WHERE data ?| ARRAY['navigation', 'navigationId', 'navigationIdentity'];

-- +goose Down
-- Откат восстанавливает структуру; удалённые связи восстановить невозможно.
ALTER TABLE projects ADD COLUMN navigation_id UUID;
ALTER TABLE projects
    ADD CONSTRAINT projects_navigation_fk
    FOREIGN KEY (workspace_id, navigation_id) REFERENCES navigations (workspace_id, id);
