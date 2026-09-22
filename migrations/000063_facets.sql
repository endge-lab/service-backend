-- +goose Up
CREATE TABLE facets
(
    id           UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id UUID        NOT NULL,
    identity     TEXT        NOT NULL,
    display_name TEXT        NOT NULL,
    icon         TEXT        NOT NULL,
    color        TEXT        NOT NULL,
    position     INTEGER     NOT NULL CHECK (position >= 0),
    meta         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    active       BOOLEAN     NOT NULL DEFAULT TRUE,
    deleted_at   TIMESTAMPTZ,
    created_by   UUID        NOT NULL,
    updated_by   UUID        NOT NULL,
    revision     INTEGER     NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, identity),
    UNIQUE (workspace_id, id),
    CONSTRAINT facets_workspace_fk FOREIGN KEY (workspace_id) REFERENCES workspaces (id),
    CONSTRAINT facets_created_by_fk FOREIGN KEY (created_by) REFERENCES service_users (id),
    CONSTRAINT facets_updated_by_fk FOREIGN KEY (updated_by) REFERENCES service_users (id)
);

CREATE UNIQUE INDEX facets_active_position_key
    ON facets (workspace_id, position)
    WHERE deleted_at IS NULL;

CREATE TABLE facet_documents
(
    id            UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    workspace_id  UUID        NOT NULL,
    facet_id      UUID        NOT NULL,
    identity      TEXT        NOT NULL,
    display_name  TEXT        NOT NULL,
    description   TEXT,
    configuration JSONB       NOT NULL DEFAULT '{"mode":"inherit","patch":{}}'::jsonb,
    meta          JSONB       NOT NULL DEFAULT '{}'::jsonb,
    active        BOOLEAN     NOT NULL DEFAULT TRUE,
    deleted_at    TIMESTAMPTZ,
    created_by    UUID        NOT NULL,
    updated_by    UUID        NOT NULL,
    revision      INTEGER     NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (facet_id, identity),
    UNIQUE (workspace_id, id),
    CONSTRAINT facet_documents_facet_fk FOREIGN KEY (workspace_id, facet_id) REFERENCES facets (workspace_id, id),
    CONSTRAINT facet_documents_workspace_fk FOREIGN KEY (workspace_id) REFERENCES workspaces (id),
    CONSTRAINT facet_documents_created_by_fk FOREIGN KEY (created_by) REFERENCES service_users (id),
    CONSTRAINT facet_documents_updated_by_fk FOREIGN KEY (updated_by) REFERENCES service_users (id)
);

CREATE INDEX facet_documents_list_idx
    ON facet_documents (workspace_id, facet_id, deleted_at, identity);

-- +goose Down
DROP TABLE IF EXISTS facet_documents;
DROP TABLE IF EXISTS facets;
