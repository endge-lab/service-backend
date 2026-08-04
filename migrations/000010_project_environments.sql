-- +goose Up
CREATE TABLE project_environments
(
    workspace_id   UUID    NOT NULL,
    project_id     UUID    NOT NULL,
    environment_id UUID    NOT NULL,
    sort_order     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, project_id, environment_id),
    FOREIGN KEY (workspace_id, project_id) REFERENCES projects (workspace_id, id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id, environment_id) REFERENCES environments (workspace_id, id)
);

-- +goose Down
DROP TABLE IF EXISTS project_environments;
