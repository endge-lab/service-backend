-- +goose Up
ALTER TABLE workspaces
    ALTER COLUMN configuration SET DEFAULT '{
      "vars": [],
      "locales": [
        {"code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr"},
        {"code": "en", "displayName": "English", "shortLabel": "EN", "direction": "ltr"}
      ],
      "defaultLocale": "en",
      "fallbackLocale": "en",
      "themes": [
        {"identity": "light", "displayName": "Светлая"},
        {"identity": "dark", "displayName": "Тёмная"}
      ],
      "defaultTheme": "dark",
      "timezones": [
        {"identity": "local", "displayName": "Локальное время"},
        {"identity": "UTC", "displayName": "UTC"}
      ],
      "defaultTimezone": "local",
      "defaultAuthProfileIdentity": null,
      "sfcAdapterIds": ["vue-native"],
      "defaultSfcAdapterId": "vue-native",
      "sfcEditing": {
        "cancelOn": [
          {"event": "keydown", "key": ["Escape"], "prevent": true, "stop": true},
          {"event": "focusout"}
        ],
        "commitOn": [{"event": "keydown", "key": ["Enter"], "prevent": true}]
      },
      "diagnostics": {},
      "values": {}
    }'::jsonb;

-- Дополняем отсутствующие поля, сохраняя все явно заданные значения.
-- Исторические snapshots не изменяются.
WITH defaults(configuration) AS (
    VALUES ('{
      "vars": [],
      "locales": [
        {"code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr"},
        {"code": "en", "displayName": "English", "shortLabel": "EN", "direction": "ltr"}
      ],
      "defaultLocale": "en",
      "fallbackLocale": "en",
      "themes": [
        {"identity": "light", "displayName": "Светлая"},
        {"identity": "dark", "displayName": "Тёмная"}
      ],
      "defaultTheme": "dark",
      "timezones": [
        {"identity": "local", "displayName": "Локальное время"},
        {"identity": "UTC", "displayName": "UTC"}
      ],
      "defaultTimezone": "local",
      "defaultAuthProfileIdentity": null,
      "sfcAdapterIds": ["vue-native"],
      "defaultSfcAdapterId": "vue-native",
      "sfcEditing": {
        "cancelOn": [
          {"event": "keydown", "key": ["Escape"], "prevent": true, "stop": true},
          {"event": "focusout"}
        ],
        "commitOn": [{"event": "keydown", "key": ["Enter"], "prevent": true}]
      },
      "diagnostics": {},
      "values": {}
    }'::jsonb)
)
UPDATE workspaces w
SET configuration = defaults.configuration || w.configuration,
    updated_at = NOW(),
    revision = w.revision + 1
FROM defaults
WHERE jsonb_typeof(w.configuration) = 'object'
  AND w.configuration IS DISTINCT FROM defaults.configuration || w.configuration;

-- +goose Down
ALTER TABLE workspaces
    ALTER COLUMN configuration SET DEFAULT '{
      "sfcEditing": {
        "cancelOn": [
          {"event": "keydown", "key": ["Escape"], "prevent": true, "stop": true},
          {"event": "focusout"}
        ],
        "commitOn": [{"event": "keydown", "key": ["Enter"], "prevent": true}]
      }
    }'::jsonb;

-- Сохранённые настройки не удаляем: пользователь мог изменить их после миграции.
SELECT 1;
