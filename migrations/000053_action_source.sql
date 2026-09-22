-- +goose Up
UPDATE actions
SET data = (
        data
        - 'definition'
        - 'input'
        - 'output'
    ) || jsonb_build_object(
        'source', E'defineAction({\n  contract: {\n    input: field(''Object''),\n    output: field(''Object''),\n  },\n\n  steps: {\n    result: input(),\n  },\n\n  output: output(''result''),\n})\n',
        'sourceVersion', 1,
        'defaultImplementation', jsonb_build_object('kind', 'source')
    ),
    updated_at = NOW(),
    revision = revision + 1
WHERE NOT (data ? 'source')
   OR NULLIF(BTRIM(data ->> 'source'), '') IS NULL;

-- +goose Down
-- Source is retained because it can contain user-authored changes after migration.
SELECT 1;
