-- +goose Up
-- +goose StatementBegin
CREATE TABLE genre_alias
(
    id             varchar(255) not null primary key,
    alias_name     varchar(255) not null unique,
    canonical_name varchar(255) not null,
    created_at     datetime,
    updated_at     datetime
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX genre_alias_canonical_name on genre_alias (canonical_name);
-- +goose StatementEnd

-- +goose Down
