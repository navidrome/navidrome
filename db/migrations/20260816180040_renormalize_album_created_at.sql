-- +goose Up

-- Repairs album.created_at values stored in RFC3339 T-format by CopyAttributes.
-- These values sort incorrectly in "Recently Added", which compares timestamps as raw strings.

UPDATE album SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00')
WHERE created_at LIKE '%T%';

-- +goose Down

SELECT 1;
