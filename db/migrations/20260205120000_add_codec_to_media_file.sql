-- +goose Up
-- +goose StatementBegin
ALTER TABLE media_file ADD COLUMN codec VARCHAR(255) DEFAULT '' NOT NULL;
CREATE INDEX IF NOT EXISTS media_file_codec ON media_file(codec);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS media_file_codec;
ALTER TABLE media_file DROP COLUMN codec;
-- +goose StatementEnd
