-- +goose Up
-- Интерактивный default profile сохраняет identity/id, но переходит на общий OIDC contract.
UPDATE auth_profiles
SET display_name = 'OIDC (Default)',
    data = jsonb_build_object(
        'adapterId', 'oidc',
        'config', jsonb_build_object(
            'issuer', '{OIDC_ISSUER}',
            'clientId', COALESCE(NULLIF(data #>> '{config,clientId}', ''), 'endge-configurator'),
            'scopes', jsonb_build_array('openid', 'profile')
        ),
        'credentials', '{}'::jsonb,
        'session', jsonb_build_object(
            'storage', 'memory',
            'persistRefreshToken', false
        )
    ),
    active = true,
    updated_at = NOW(),
    revision = revision + 1
WHERE identity = 'keycloak-default'
  AND deleted_at IS NULL;

-- Password-grant profiles не имеют безопасного эквивалента в новом contract.
UPDATE auth_profiles
SET data = jsonb_build_object(
        'adapterId', 'oidc',
        'config', '{}'::jsonb,
        'credentials', '{}'::jsonb
    ),
    active = false,
    deleted_at = COALESCE(deleted_at, NOW()),
    updated_at = NOW(),
    revision = revision + 1
WHERE data ->> 'adapterId' = 'keycloak'
  AND identity <> 'keycloak-default';

-- +goose Down
UPDATE auth_profiles
SET display_name = 'Keycloak (Form)',
    data = jsonb_build_object(
        'adapterId', 'keycloak',
        'config', jsonb_build_object(
            'loginMode', 'interactive',
            'baseUrl', '{ENDPOINT_AUTH}',
            'clientId', COALESCE(NULLIF(data #>> '{config,clientId}', ''), 'endge-configurator'),
            'scope', 'openid profile',
            'tokenPath', '/token',
            'logoutPath', '/logout',
            'userinfoPath', '/userinfo',
            'refreshSkewMs', 300000
        ),
        'credentialRefs', '{}'::jsonb,
        'persist', 'localStorage'
    ),
    active = false,
    updated_at = NOW(),
    revision = revision + 1
WHERE identity = 'keycloak-default'
  AND data ->> 'adapterId' = 'oidc';
