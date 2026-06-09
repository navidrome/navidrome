-- +goose Up
-- +goose StatementBegin
CREATE TABLE scrobbles(
    media_file_id VARCHAR(255) NOT NULL
        REFERENCES media_file(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    user_id VARCHAR(255) NOT NULL 
        REFERENCES user(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    submission_time INTEGER NOT NULL
);
CREATE INDEX scrobbles_date ON scrobbles (submission_time);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE scrobbles;
-- +goose StatementEnd
