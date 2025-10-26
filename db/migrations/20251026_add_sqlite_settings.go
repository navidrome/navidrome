-- +migrate Up
-- Enable WAL mode and set busy timeout if not already set
UPDATE user_property SET value = 'WAL' 
WHERE name = 'sqlite_journal_mode' AND NOT EXISTS (
    SELECT 1 FROM user_property WHERE name = 'sqlite_journal_mode'
);

INSERT INTO user_property (name, value) 
SELECT 'sqlite_busy_timeout', '5000'
WHERE NOT EXISTS (
    SELECT 1 FROM user_property WHERE name = 'sqlite_busy_timeout'
);

INSERT INTO user_property (name, value) 
SELECT 'sqlite_sync_mode', 'NORMAL'
WHERE NOT EXISTS (
    SELECT 1 FROM user_property WHERE name = 'sqlite_sync_mode'
);

-- +migrate Down
DELETE FROM user_property WHERE name IN (
    'sqlite_journal_mode',
    'sqlite_busy_timeout',
    'sqlite_sync_mode'
);