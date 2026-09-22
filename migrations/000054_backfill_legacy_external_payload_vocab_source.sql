-- +goose Up
-- Исправляет только автоматически созданный provider-less Source legacy Vocab.
WITH legacy AS (
    SELECT id,
           COALESCE(data ->> 'baseApiUrl', '') AS base_api_url,
           COALESCE(NULLIF(data ->> 'collectionSlug', ''), identity) AS collection_slug,
           CASE
               WHEN data ->> 'authMode' IN ('inherit', 'profile', 'none') THEN data ->> 'authMode'
               ELSE 'inherit'
           END AS auth_mode,
           COALESCE(data ->> 'authProfileIdentity', '') AS auth_profile_identity
    FROM vocabs
    WHERE data ->> 'mode' = 'external_payload'
      AND (
          NOT (data ? 'source')
          OR NULLIF(BTRIM(data ->> 'source'), '') IS NULL
          OR data ->> 'source' IN (
              E'defineVocab({\n  outputs: {\n    items: output().from(response()),\n  },\n})',
              E'defineVocab({\n  outputs: {\n    items: output()\n      .from(response()),\n  },\n})',
              'defineVocab({ outputs: { items: output().from(response()) } })'
          )
      )
), generated AS (
    SELECT id,
           E'defineVocab({\n  provider: payload({\n    baseUrl: '
           || CASE
                  WHEN base_api_url ~ '^\{[A-Za-z_][A-Za-z0-9_]*\}$'
                      THEN 'env(' || to_jsonb(SUBSTRING(base_api_url FROM 2 FOR LENGTH(base_api_url) - 2))::text || ')'
                  ELSE to_jsonb(base_api_url)::text
              END
           || E',\n    collection: ' || to_jsonb(collection_slug)::text
           || E',\n    auth: { mode: ' || to_jsonb(auth_mode)::text
           || CASE
                  WHEN auth_mode = 'profile'
                      THEN ', profile: ' || to_jsonb(auth_profile_identity)::text
                  ELSE ''
              END
           || E' },\n  }),\n\n  outputs: {\n    items: output()\n      .from(response()),\n  },\n})\n' AS source
    FROM legacy
)
UPDATE vocabs AS vocab
SET data = jsonb_set(
        jsonb_set(vocab.data, '{source}', to_jsonb(generated.source), true),
        '{sourceVersion}',
        '1'::jsonb,
        true
    ),
    updated_at = NOW(),
    revision = revision + 1
FROM generated
WHERE vocab.id = generated.id;

-- +goose Down
SELECT 1;
