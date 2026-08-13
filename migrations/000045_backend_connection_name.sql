-- +goose Up
ALTER TABLE backend_connections
    ADD COLUMN name TEXT;

UPDATE backend_connections
SET name = base_url
WHERE name IS NULL OR BTRIM(name) = '';

ALTER TABLE backend_connections
    ALTER COLUMN name SET NOT NULL,
    ADD CONSTRAINT backend_connections_name_check CHECK (CHAR_LENGTH(BTRIM(name)) BETWEEN 1 AND 160);

-- +goose Down
ALTER TABLE backend_connections
    DROP CONSTRAINT IF EXISTS backend_connections_name_check,
    DROP COLUMN IF EXISTS name;
