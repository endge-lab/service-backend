-- +goose Up
CREATE TABLE backend_connections
(
    id         UUID        PRIMARY KEY,
    base_url   TEXT        NOT NULL UNIQUE,
    created_by UUID        NOT NULL REFERENCES service_users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS backend_connections;
