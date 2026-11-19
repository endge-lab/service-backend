-- +goose Up
ALTER TABLE modules ADD COLUMN deleted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE modules DROP COLUMN deleted_at;
