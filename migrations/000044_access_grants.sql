-- +goose Up
ALTER TABLE service_users
    DROP CONSTRAINT service_users_provider_subject_key;

ALTER TABLE service_users
    ADD CONSTRAINT service_users_provider_issuer_subject_key UNIQUE (provider_id, issuer, subject);

CREATE INDEX service_users_active_username_prefix_idx
    ON service_users (LOWER(username) text_pattern_ops, id)
    WHERE active = TRUE AND is_system = FALSE;

CREATE TABLE access_grants
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES service_users (id) ON DELETE CASCADE,
    scope_type    TEXT        NOT NULL CHECK (scope_type IN ('platform', 'workspace')),
    workspace_id  UUID REFERENCES workspaces (id) ON DELETE CASCADE,
    role          TEXT        NOT NULL CHECK (role IN ('viewer', 'editor', 'admin')),
    created_by    UUID        NOT NULL REFERENCES service_users (id),
    updated_by    UUID        NOT NULL REFERENCES service_users (id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT access_grants_scope_role_check CHECK (
        (scope_type = 'platform' AND workspace_id IS NULL AND role = 'admin') OR
        (scope_type = 'workspace' AND workspace_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX access_grants_platform_user_key
    ON access_grants (user_id)
    WHERE scope_type = 'platform';

CREATE UNIQUE INDEX access_grants_workspace_user_key
    ON access_grants (workspace_id, user_id)
    WHERE scope_type = 'workspace';

CREATE INDEX access_grants_user_scope_idx ON access_grants (user_id, scope_type, workspace_id);
CREATE INDEX access_grants_workspace_idx ON access_grants (workspace_id, user_id);

INSERT INTO access_grants
    (user_id, scope_type, workspace_id, role, created_by, updated_by, created_at, updated_at)
SELECT user_id, 'workspace', workspace_id, role, created_by, created_by, created_at, updated_at
FROM workspace_memberships;

-- Сохраняем прежний эффективный editor-доступ к default явными grants.
INSERT INTO access_grants
    (user_id, scope_type, workspace_id, role, created_by, updated_by)
SELECT u.id, 'workspace', w.id, 'editor', u.id, u.id
FROM service_users u
CROSS JOIN workspaces w
WHERE u.active = TRUE
  AND u.is_system = FALSE
  AND w.identity = 'default'
  AND NOT EXISTS (
      SELECT 1 FROM access_grants g
      WHERE g.scope_type = 'workspace' AND g.workspace_id = w.id AND g.user_id = u.id
  );

-- Переносим только действующие legacy Platform Admin sessions.
INSERT INTO access_grants
    (user_id, scope_type, role, created_by, updated_by)
SELECT DISTINCT u.id, 'platform', 'admin', u.id, u.id
FROM service_users u
JOIN configurator_auth_sessions s
  ON s.provider_id = u.provider_id AND s.issuer = u.issuer AND s.subject = u.subject
WHERE u.active = TRUE
  AND u.is_system = FALSE
  AND s.platform_admin = TRUE
  AND s.revoked_at IS NULL
  AND s.expires_at > NOW()
  AND NOT EXISTS (
      SELECT 1 FROM access_grants g
      WHERE g.scope_type = 'platform' AND g.user_id = u.id
  );

-- +goose Down
DROP TABLE IF EXISTS access_grants;
DROP INDEX IF EXISTS service_users_active_username_prefix_idx;

ALTER TABLE service_users
    DROP CONSTRAINT service_users_provider_issuer_subject_key;

ALTER TABLE service_users
    ADD CONSTRAINT service_users_provider_subject_key UNIQUE (provider_id, subject);
