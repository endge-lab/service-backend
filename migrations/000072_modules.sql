-- +goose Up
CREATE TABLE modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identity TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_by UUID NOT NULL REFERENCES service_users (id)
);

CREATE TABLE module_versions (
    module_id UUID NOT NULL REFERENCES modules (id),
    version TEXT NOT NULL,
    manifest JSONB NOT NULL,
    has_css BOOLEAN NOT NULL,
    archive BYTEA NOT NULL,
    sha256 TEXT NOT NULL CHECK (char_length(sha256) = 64),
    size INTEGER NOT NULL CHECK (size = octet_length(archive)),
    published_by UUID NOT NULL REFERENCES service_users (id),
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (module_id, version)
);

-- +goose Down
DROP TABLE module_versions;
DROP TABLE modules;
