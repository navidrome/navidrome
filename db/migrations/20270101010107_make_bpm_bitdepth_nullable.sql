-- +goose Up
drop index if exists media_file_bpm;

alter table media_file add column bpm_new integer;
alter table media_file add column bit_depth_new integer;

update media_file set
    bpm_new = nullif(bpm, 0),
    bit_depth_new = nullif(bit_depth, 0);

alter table media_file drop column bpm;
alter table media_file drop column bit_depth;

alter table media_file rename column bpm_new to bpm;
alter table media_file rename column bit_depth_new to bit_depth;

create index if not exists media_file_bpm on media_file (bpm);

-- +goose Down
drop index if exists media_file_bpm;

alter table media_file add column bpm_old integer default 0 not null;
alter table media_file add column bit_depth_old integer default 0 not null;

update media_file set
    bpm_old = coalesce(bpm, 0),
    bit_depth_old = coalesce(bit_depth, 0);

alter table media_file drop column bpm;
alter table media_file drop column bit_depth;

alter table media_file rename column bpm_old to bpm;
alter table media_file rename column bit_depth_old to bit_depth;

create index if not exists media_file_bpm on media_file (bpm);
