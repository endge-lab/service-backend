-- +goose Up
CREATE TABLE mutation_batches
(
    id            UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id  UUID REFERENCES workspaces (id),
    kind          TEXT        NOT NULL,
    actor_user_id UUID        NOT NULL REFERENCES service_users (id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS mutation_batches;
