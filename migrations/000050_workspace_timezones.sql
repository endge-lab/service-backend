-- +goose Up
UPDATE workspaces
SET configuration = configuration
  || CASE
    WHEN configuration ? 'timezones' THEN '{}'::jsonb
    ELSE jsonb_build_object('timezones', '[{"identity":"local","displayName":"Локальное время"},{"identity":"UTC","displayName":"UTC"}]'::jsonb)
  END
  || CASE
    WHEN configuration ? 'defaultTimezone' THEN '{}'::jsonb
    ELSE jsonb_build_object('defaultTimezone', 'local')
  END
WHERE NOT (configuration ? 'timezones')
   OR NOT (configuration ? 'defaultTimezone');

-- +goose Down
UPDATE workspaces
SET configuration = configuration - 'timezones' - 'defaultTimezone';
