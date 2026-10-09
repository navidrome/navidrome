-- +goose Up
-- +goose StatementBegin
-- 32-bit builds cannot read values above the int32 range written by 64-bit builds.
update media_file set track_number = 0
where track_number < 0 or track_number > 2147483647;

update media_file set disc_number = 0
where disc_number < 0 or disc_number > 2147483647;

update media_file set bpm = null
where bpm < 0 or bpm > 2147483647;

update album set discs = (
	select json_group_object(key, value) from json_each(album.discs)
	where cast(key as integer) between 0 and 2147483647
)
where json_valid(discs) and exists (
	select 1 from json_each(album.discs)
	where cast(key as integer) not between 0 and 2147483647
);
-- +goose StatementEnd

-- +goose Down
SELECT 1;
