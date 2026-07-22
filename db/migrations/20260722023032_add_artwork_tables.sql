-- +goose Up
CREATE TABLE artwork (
  hash TEXT PRIMARY KEY,
  mime TEXT NOT NULL,
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  blur_hash TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE item_artwork (
  item_kind TEXT NOT NULL,
  item_id TEXT NOT NULL,
  image_type TEXT NOT NULL DEFAULT 'primary',
  hash TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  source_path TEXT NOT NULL DEFAULT '',
  ref_mtime INTEGER NOT NULL DEFAULT 0,
  attempted_at TIMESTAMP,
  updated_at TIMESTAMP,
  PRIMARY KEY (item_kind, item_id, image_type)
) WITHOUT ROWID;
CREATE INDEX ix_item_artwork_hash ON item_artwork(hash);

CREATE TABLE artwork_queue (
  item_kind TEXT NOT NULL,
  item_id TEXT NOT NULL,
  image_type TEXT NOT NULL DEFAULT 'primary',
  priority INTEGER NOT NULL DEFAULT 0,
  attempts INTEGER NOT NULL DEFAULT 0,
  retry_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  enqueued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (item_kind, item_id, image_type)
) WITHOUT ROWID;
-- Ordered to match DequeueBatch (priority DESC, enqueued_at) so drains stop after n rows; retry_at makes it covering.
CREATE INDEX ix_artwork_queue_drain ON artwork_queue(priority DESC, enqueued_at, retry_at);

-- +goose Down
DROP TABLE artwork_queue;
DROP TABLE item_artwork;
DROP TABLE artwork;
