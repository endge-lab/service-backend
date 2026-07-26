-- +goose Up
ALTER TABLE types
    ADD COLUMN source TEXT NOT NULL DEFAULT '',
    ADD COLUMN source_version INTEGER NOT NULL DEFAULT 1;

CREATE OR REPLACE FUNCTION pg_temp.endge_type_source_from_schema(schema_value JSONB)
RETURNS TEXT
LANGUAGE plpgsql
AS $$
DECLARE
    fields_value JSONB;
    body TEXT;
BEGIN
    fields_value := schema_value -> 'fields';

    IF jsonb_typeof(fields_value) <> 'array' THEN
        RAISE EXCEPTION 'Cannot migrate Type schema without an array fields property: %', schema_value;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM jsonb_array_elements(fields_value) AS entries(field)
        WHERE NULLIF(BTRIM(entries.field ->> 'name'), '') IS NULL
           OR NULLIF(BTRIM(entries.field ->> 'type'), '') IS NULL
           OR (
                jsonb_typeof(entries.field -> 'params') = 'array'
                AND jsonb_array_length(entries.field -> 'params') > 0
           )
           OR (
                jsonb_typeof(entries.field -> 'params') = 'object'
                AND entries.field -> 'params' <> '{}'::jsonb
           )
           OR (
                jsonb_typeof(entries.field -> 'runtime-filters') = 'array'
                AND jsonb_array_length(entries.field -> 'runtime-filters') > 0
           )
           OR (
                jsonb_typeof(entries.field -> 'runtimeFilters') = 'array'
                AND jsonb_array_length(entries.field -> 'runtimeFilters') > 0
           )
    ) THEN
        RAISE EXCEPTION 'Cannot migrate Type schema with unnamed, untyped, method or runtime-filter fields: %', schema_value;
    END IF;

    IF jsonb_array_length(fields_value) = 0 THEN
        RETURN 'defineType({})' || E'\n';
    END IF;

    SELECT string_agg(
        '  '
            || to_jsonb(entries.field ->> 'name')::text
            || ': field(type('
            || to_jsonb(entries.field ->> 'type')::text
            || '))'
            || CASE
                WHEN COALESCE((entries.field ->> 'isArray')::boolean, false)
                    THEN E'\n    .array()'
                ELSE ''
            END
            || CASE
                WHEN COALESCE((entries.field ->> 'optional')::boolean, false)
                    THEN E'\n    .optional()'
                ELSE ''
            END
            || ',',
        E'\n\n'
        ORDER BY entries.ordinality
    )
    INTO body
    FROM jsonb_array_elements(fields_value) WITH ORDINALITY AS entries(field, ordinality);

    RETURN 'defineType({' || E'\n' || body || E'\n})\n';
END;
$$;

UPDATE types
SET source = pg_temp.endge_type_source_from_schema(schema)
WHERE is_primitive = FALSE
  AND BTRIM(source) = '';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM types
        WHERE is_primitive = FALSE
          AND BTRIM(source) = ''
    ) THEN
        RAISE EXCEPTION 'Type Source migration left user types without source.';
    END IF;
END;
$$;

ALTER TABLE types DROP COLUMN schema;

-- +goose Down
ALTER TABLE types
    ADD COLUMN schema JSONB NOT NULL DEFAULT '{}'::jsonb;
