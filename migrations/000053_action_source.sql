-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM actions
        WHERE (NOT (data ? 'source') OR NULLIF(BTRIM(data ->> 'source'), '') IS NULL)
          AND (
              CASE WHEN jsonb_typeof(data -> 'definition' -> 'nodes') = 'array'
                  THEN jsonb_array_length(data -> 'definition' -> 'nodes') ELSE 0 END > 0
              OR CASE WHEN jsonb_typeof(data -> 'definition' -> 'edges') = 'array'
                  THEN jsonb_array_length(data -> 'definition' -> 'edges') ELSE 0 END > 0
          )
    ) THEN
        RAISE EXCEPTION 'Action Source migration stopped: non-empty legacy Flow documents must be migrated manually';
    END IF;
END $$;
-- +goose StatementEnd

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
