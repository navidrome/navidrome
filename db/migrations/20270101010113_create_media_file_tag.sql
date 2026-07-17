-- +goose Up
-- +goose StatementBegin
CREATE TABLE media_file_tag
(
	user_id       varchar(255) not null
		REFERENCES user(id)
			ON DELETE CASCADE
			ON UPDATE CASCADE,
	media_file_id varchar(255) not null,
	tag_name      varchar(255) not null,
	created_at    datetime,
	unique (user_id, media_file_id, tag_name)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX media_file_tag_tag_name on media_file_tag (tag_name);
-- +goose StatementEnd

-- +goose Down
