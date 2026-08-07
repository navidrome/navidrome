-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_feature_permission
(
	user_id    varchar(255) not null
		REFERENCES user(id)
			ON DELETE CASCADE
			ON UPDATE CASCADE,
	feature    varchar(255) not null,
	enabled    bool         not null default true,
	updated_at datetime,
	primary key (user_id, feature)
);
-- +goose StatementEnd

-- +goose Down
