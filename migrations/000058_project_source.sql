-- +goose Up
-- Старые композиции остаются самостоятельными документами для ручного переноса.
UPDATE projects
SET data = jsonb_build_object(
    'source', E'defineComposition({\n  activateOn: startup(),\n  data: {},\n  resources: {},\n  runtimes: {},\n  hooks: [],\n  outputs: {},\n})\n',
    'sourceVersion', 1
) || data
WHERE NOT (data ? 'source') OR NOT (data ? 'sourceVersion');

-- +goose Down
-- Source проекта сохраняется при откате: удаление authored-данных необратимо.
SELECT 1;
