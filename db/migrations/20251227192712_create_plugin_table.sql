-- +goose Up
CREATE TABLE IF NOT EXISTS plugin (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    manifest JSONB NOT NULL,
    config JSONB,
    users JSONB,
    all_users BOOL NOT NULL DEFAULT false,
    libraries JSONB,
    all_libraries BOOL NOT NULL DEFAULT false,
    enabled BOOL NOT NULL DEFAULT false,
    last_error TEXT,
    sha256 TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS plugin;
