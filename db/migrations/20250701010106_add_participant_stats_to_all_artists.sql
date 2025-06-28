-- +goose Up
-- +goose StatementBegin
WITH artist_role_counters AS (
    SELECT jt.atom AS artist_id,
            substr(
                    replace(jt.path, '$.', ''),
                    1,
                    CASE WHEN instr(replace(jt.path, '$.', ''), '[') > 0
                            THEN instr(replace(jt.path, '$.', ''), '[') - 1
                        ELSE length(replace(jt.path, '$.', ''))
                        END
            ) AS role,
            count(DISTINCT mf.album_id) AS album_count,
            count(mf.id) AS count,
            sum(mf.size) AS size
    FROM media_file mf
    JOIN json_tree(mf.participants) jt ON jt.key = 'id' AND jt.atom IS NOT NULL
    GROUP BY jt.atom, role
),
artist_total_counters AS (
    SELECT mfa.artist_id,
            'total' AS role,
            count(DISTINCT mf.album_id) AS album_count,
            count(DISTINCT mf.id) AS count,
            sum(mf.size) AS size
    FROM media_file_artists mfa
    JOIN media_file mf ON mfa.media_file_id = mf.id
    GROUP BY mfa.artist_id
),
artist_participant_counter AS (
    SELECT mfa.artist_id,
        'maincredit' AS role,
        count(DISTINCT mf.album_id) AS album_count,
        count(DISTINCT mf.id) AS count,
        sum(mf.size) AS size
    FROM media_file_artists mfa
    JOIN media_file mf ON mfa.media_file_id = mf.id
    AND mfa.role IN ('albumartist', 'artist')
    GROUP BY mfa.artist_id
),
combined_counters AS (
    SELECT artist_id, role, album_count, count, size FROM artist_role_counters
    UNION
    SELECT artist_id, role, album_count, count, size FROM artist_total_counters
    UNION
    SELECT artist_id, role, album_count, count, size FROM artist_participant_counter
),
artist_counters AS (
    SELECT artist_id AS id,
            json_group_object(
                    replace(role, '"', ''),
                    json_object('a', album_count, 'm', count, 's', size)
            ) AS counters
    FROM combined_counters
    GROUP BY artist_id
)
UPDATE artist
SET stats = coalesce((SELECT counters FROM artist_counters ac WHERE ac.id = artist.id), '{}'),
    updated_at = datetime(current_timestamp, 'localtime')
WHERE artist.id <> '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
