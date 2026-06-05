-- +goose Up
-- +goose StatementBegin
CREATE TABLE share_tmp
(
    id              varchar(255) not null
        primary key,
    expires_at      datetime,
    last_visited_at datetime,
    resource_ids    varchar      not null,
    created_at      datetime,
    updated_at      datetime,
    user_id         varchar(255) not null
        constraint share_user_id_fk
            references user
                on update cascade on delete cascade,
    downloadable bool not null default false,
    description varchar not null default '',
    resource_type varchar not null default '',
    contents varchar not null default '',
    format varchar not null default '',
    max_bit_rate integer not null default 0,
    visit_count integer not null default 0
);


INSERT INTO share_tmp(
    id, expires_at, last_visited_at, resource_ids, created_at, updated_at, user_id, downloadable, description, resource_type, contents, format, max_bit_rate, visit_count
) SELECT id, expires_at, last_visited_at, resource_ids, created_at, updated_at, user_id, downloadable, description, resource_type, contents, format, max_bit_rate, visit_count 
FROM share;

DROP TABLE share;

ALTER TABLE share_tmp RENAME To share;
-- +goose StatementEnd

-- +goose Down
