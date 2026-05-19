-- +goose Up
-- +goose StatementBegin
CREATE TABLE playlist_permissions
(
    playlist_id VARCHAR(255) NOT NULL,
    user_id     VARCHAR(255) NOT NULL,
    permission  VARCHAR(10) NOT NULL,
    PRIMARY KEY (playlist_id, user_id),
    FOREIGN KEY (playlist_id) REFERENCES playlist(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
)
-- +goose StatementEnd

-- +goose Down
