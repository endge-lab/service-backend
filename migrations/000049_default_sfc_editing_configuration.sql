-- +goose Up
ALTER TABLE workspaces
    ALTER COLUMN configuration SET DEFAULT '{
      "sfcEditing": {
        "cancelOn": [
          {"event": "keydown", "key": ["Escape"], "prevent": true, "stop": true},
          {"event": "focusout"}
        ],
        "commitOn": [
          {"event": "keydown", "key": ["Enter"], "prevent": true}
        ]
      }
    }'::jsonb;

UPDATE workspaces
SET configuration = jsonb_set(
        configuration,
        '{sfcEditing}',
        '{
          "cancelOn": [
            {"event": "keydown", "key": ["Escape"], "prevent": true, "stop": true},
            {"event": "focusout"}
          ],
          "commitOn": [
            {"event": "keydown", "key": ["Enter"], "prevent": true}
          ]
        }'::jsonb,
        true
    ),
    updated_at = NOW(),
    revision = revision + 1
WHERE NOT (configuration ? 'sfcEditing');

UPDATE projects
SET data = jsonb_set(
        data,
        '{configuration,value,sfcEditing}',
        '{
          "cancelOn": [
            {"event": "keydown", "key": ["Escape"], "prevent": true, "stop": true},
            {"event": "focusout"}
          ],
          "commitOn": [
            {"event": "keydown", "key": ["Enter"], "prevent": true}
          ]
        }'::jsonb,
        true
    ),
    updated_at = NOW(),
    revision = revision + 1
WHERE data #>> '{configuration,mode}' = 'replace'
  AND jsonb_typeof(data #> '{configuration,value}') = 'object'
  AND NOT ((data #> '{configuration,value}') ? 'sfcEditing');

UPDATE tenants
SET data = jsonb_set(
        data,
        '{configuration,value,sfcEditing}',
        '{
          "cancelOn": [
            {"event": "keydown", "key": ["Escape"], "prevent": true, "stop": true},
            {"event": "focusout"}
          ],
          "commitOn": [
            {"event": "keydown", "key": ["Enter"], "prevent": true}
          ]
        }'::jsonb,
        true
    ),
    updated_at = NOW(),
    revision = revision + 1
WHERE data #>> '{configuration,mode}' = 'replace'
  AND jsonb_typeof(data #> '{configuration,value}') = 'object'
  AND NOT ((data #> '{configuration,value}') ? 'sfcEditing');

UPDATE environments
SET data = jsonb_set(
        data,
        '{configuration,value,sfcEditing}',
        '{
          "cancelOn": [
            {"event": "keydown", "key": ["Escape"], "prevent": true, "stop": true},
            {"event": "focusout"}
          ],
          "commitOn": [
            {"event": "keydown", "key": ["Enter"], "prevent": true}
          ]
        }'::jsonb,
        true
    ),
    updated_at = NOW(),
    revision = revision + 1
WHERE data #>> '{configuration,mode}' = 'replace'
  AND jsonb_typeof(data #> '{configuration,value}') = 'object'
  AND NOT ((data #> '{configuration,value}') ? 'sfcEditing');

-- +goose Down
ALTER TABLE workspaces
    ALTER COLUMN configuration SET DEFAULT '{}'::jsonb;

-- Persisted values are intentionally retained: they may have been changed after migration.
SELECT 1;
