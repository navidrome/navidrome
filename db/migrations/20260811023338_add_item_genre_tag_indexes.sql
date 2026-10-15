-- +goose Up
create table if not exists media_file_tags(
    media_file_id varchar not null
        references media_file (id) on delete cascade,
    tag_id varchar not null
        references tag (id) on delete cascade,
    constraint media_file_tags unique (media_file_id, tag_id)
);
create index if not exists media_file_tags_tag_id on media_file_tags (tag_id);

create table if not exists album_tags(
    album_id varchar not null
        references album (id) on delete cascade,
    tag_id varchar not null
        references tag (id) on delete cascade,
    constraint album_tags unique (album_id, tag_id)
);
create index if not exists album_tags_tag_id on album_tags (tag_id);

-- Backfill genre rows from the per-row `tags` JSON. json_tree over the "$.genre" subtree yields one
-- row per node; the "id" key nodes carry the tag ids. Single scan per table, no correlated subquery.
insert or ignore into media_file_tags (media_file_id, tag_id)
select mf.id, jt.value
from media_file mf, json_tree(mf.tags, '$.genre') jt
where jt.key = 'id' and jt.atom is not null;

insert or ignore into album_tags (album_id, tag_id)
select al.id, jt.value
from album al, json_tree(al.tags, '$.genre') jt
where jt.key = 'id' and jt.atom is not null;

-- +goose Down
drop table if exists media_file_tags;
drop table if exists album_tags;
