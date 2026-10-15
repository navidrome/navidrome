-- +goose Up
-- 20260819204637 added last_failure with DEFAULT '[]', so every row already in the table got a
-- non-empty value. That is how a give-up is now told apart from a definitive "no image", which
-- would report every pre-existing absent row as failed.
UPDATE item_artwork SET last_failure = '' WHERE last_failure = '[]';

-- +goose Down
-- Irreversible: a genuine give-up and a backfilled default are indistinguishable once normalized.
SELECT 1;
