-- +goose Up
CREATE TABLE IF NOT EXISTS plugin (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    manifest TEXT NOT NULL,
    config TEXT,
    users TEXT,
    all_users INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    sha256 TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS plugin;
