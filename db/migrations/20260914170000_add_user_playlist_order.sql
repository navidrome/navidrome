-- +goose Up
CREATE TABLE user_playlist_order (
    user_id     VARCHAR(255) NOT NULL REFERENCES user(id) ON UPDATE CASCADE ON DELETE CASCADE,
    playlist_id VARCHAR(255) NOT NULL REFERENCES playlist(id) ON UPDATE CASCADE ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    PRIMARY KEY (user_id, playlist_id),
    UNIQUE (user_id, position)
);

CREATE INDEX user_playlist_order_user_position
    ON user_playlist_order (user_id, position);

-- +goose Down
DROP TABLE user_playlist_order;
