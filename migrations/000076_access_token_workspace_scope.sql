-- +goose Up
ALTER TABLE access_tokens
    ALTER COLUMN workspace_id DROP NOT NULL,
    ADD COLUMN manage_workspaces BOOLEAN NOT NULL DEFAULT FALSE,
    ADD CONSTRAINT access_tokens_manage_workspaces_scope
        CHECK (NOT manage_workspaces OR (workspace_id IS NULL AND scope = 'write'));

-- +goose Down
-- Не удаляем токены и их историю при откате. Если есть глобальные токены,
-- PostgreSQL отклонит возврат к схеме с обязательным workspace_id.
ALTER TABLE access_tokens
    ALTER COLUMN workspace_id SET NOT NULL,
    DROP CONSTRAINT access_tokens_manage_workspaces_scope,
    DROP COLUMN manage_workspaces;
