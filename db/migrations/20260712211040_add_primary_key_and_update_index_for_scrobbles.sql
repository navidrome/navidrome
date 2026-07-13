-- +goose Up
CREATE TABLE scrobbles_tmp(
    id INTEGER PRIMARY KEY,
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
INSERT INTO scrobbles_tmp SELECT ROWID, media_file_id, user_id, submission_time FROM scrobbles;

DROP INDEX scrobbles_date;
DROP TABLE scrobbles;
ALTER TABLE scrobbles_tmp RENAME TO scrobbles;
CREATE INDEX scrobbles_user_time ON scrobbles(user_id, submission_time);


-- +goose Down
CREATE TABLE scrobbles_tmp(
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
INSERT INTO scrobbles_tmp SELECT media_file_id, user_id, submission_time FROM scrobbles;

DROP INDEX scrobbles_user_time;
DROP TABLE scrobbles;
ALTER TABLE scrobbles_tmp RENAME TO scrobbles;
CREATE INDEX scrobbles_date ON scrobbles(submission_time);