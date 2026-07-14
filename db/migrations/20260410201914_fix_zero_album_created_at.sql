-- +goose Up

-- Backfill album.created_at for rows poisoned by early scanner versions or
-- propagated via CopyAttributes during metadata-driven ID changes. Prefer the
-- oldest valid birth_time from the album's media files, fall back to updated_at.
UPDATE album
SET created_at = COALESCE(
    (SELECT MIN(birth_time)
     FROM media_file
     WHERE media_file.album_id = album.id
       AND birth_time IS NOT NULL
       AND birth_time != ''
       AND birth_time NOT LIKE '0001-%'),
    updated_at
)
WHERE created_at IS NULL
   OR created_at = ''
   OR created_at LIKE '0001-%';

-- +goose Down

SELECT 1;
