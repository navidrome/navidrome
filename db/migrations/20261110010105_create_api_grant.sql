-- +goose Up
-- +goose StatementBegin
create table api_grant (
	id varchar not null primary key,
	user_id varchar not null references user(id) on delete cascade,
	name varchar not null,
	client varchar not null,
	client_version varchar not null default '',
	scopes varchar not null default '',
	provider varchar not null,
	secret_hash varchar not null unique,
	user_epoch integer not null default 0,
	created_at datetime not null,
	last_used_at datetime,
	last_used_ip varchar not null default ''
);
create index api_grant_user_id on api_grant(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table api_grant;
-- +goose StatementEnd
