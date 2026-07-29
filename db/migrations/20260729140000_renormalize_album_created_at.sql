-- +goose Up

-- Repairs album.created_at values stored in RFC3339 T-format ("2026-07-24T17:05:39.706028243Z").
-- 20260316000000_normalize_timestamps already did a one-off pass over every timestamp column, but
-- albumRepository.CopyAttributes kept re-introducing the T-format on this one column: it read the
-- value into a Go string (go-sqlite3 decodes `datetime` columns as time.Time, and database/sql then
-- formats that as RFC3339Nano) and wrote it straight back. It runs whenever an album ID changes
-- because its tags were edited, which is routine for beets/Picard users.
--
-- Since "Recently Added" compares these timestamps as raw strings (see
-- 20260629123100_recently_added_plain_indexes) and 'T' (ASCII 84) sorts above ' ' (ASCII 32),
-- every affected album stays pinned to the top of the list.

UPDATE album SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00')
WHERE created_at LIKE '%T%';

-- +goose Down

SELECT 1;
