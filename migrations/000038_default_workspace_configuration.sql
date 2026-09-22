-- +goose Up
UPDATE workspaces
SET configuration = '{
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
  "defaultAuthProfileIdentity": null,
  "sfcAdapterIds": ["vue-native"],
  "defaultSfcAdapterId": "vue-native",
  "diagnostics": {}
}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000010'
  AND identity = 'default'
  AND head_sequence = 0
  AND configuration = '{}'::jsonb;

-- +goose Down
UPDATE workspaces
SET configuration = '{}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000010'
  AND identity = 'default'
  AND head_sequence = 0
  AND configuration = '{
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
    "defaultAuthProfileIdentity": null,
    "sfcAdapterIds": ["vue-native"],
    "defaultSfcAdapterId": "vue-native",
    "diagnostics": {}
  }'::jsonb;
