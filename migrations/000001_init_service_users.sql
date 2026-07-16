-- +goose Up
CREATE TABLE IF NOT EXISTS service_users (
  id UUID PRIMARY KEY,
  auth_user_id TEXT NOT NULL UNIQUE,
  username TEXT NOT NULL DEFAULT '',
  display_name TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS service_users;
