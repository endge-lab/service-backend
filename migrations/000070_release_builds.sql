-- +goose Up
ALTER TABLE releases ADD COLUMN build_metadata JSONB;
ALTER TABLE releases ADD COLUMN compiled_bundle BYTEA;
ALTER TABLE releases ADD CONSTRAINT releases_build_pair CHECK ((build_metadata IS NULL) = (compiled_bundle IS NULL));

-- +goose Down
ALTER TABLE releases DROP CONSTRAINT releases_build_pair;
ALTER TABLE releases DROP COLUMN compiled_bundle;
ALTER TABLE releases DROP COLUMN build_metadata;
