-- +goose Up
-- +goose StatementBegin
CREATE TABLE podcast_subscription
(
	id              varchar(255) not null primary key,
	channel_id      varchar(255) not null
		references podcast_channel (id)
			on delete cascade,
	user_id         varchar(255) not null
		references user (id)
			on delete cascade,
	download_policy varchar default 'none' not null,
	retention_count integer default 0 not null,
	retention_days  integer default 0 not null,
	max_storage_mb  integer default 0 not null,
	created_at      datetime,
	updated_at      datetime,
	unique (channel_id, user_id)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_podcast_subscription_user_id on podcast_subscription (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE annotation ADD COLUMN downloaded bool default FALSE not null;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE annotation ADD COLUMN downloaded_at datetime;
-- +goose StatementEnd

-- Backfill: every existing channel becomes a subscription for whichever admin created it, using
-- the channel's own (soon-to-be-removed) download/retention settings, so nobody silently loses
-- their existing download policy when this ships. Picks the single most-recently-updated admin as
-- a reasonable default owner - there's no other record of "who subscribed to what" pre-migration,
-- since the whole point of this change is that subscriptions didn't exist yet.
-- +goose StatementBegin
INSERT INTO podcast_subscription (id, channel_id, user_id, download_policy, retention_count, retention_days, max_storage_mb, created_at, updated_at)
SELECT lower(hex(randomblob(16))), pc.id, u.id, pc.download_policy, pc.retention_count, pc.retention_days, pc.max_storage_mb, pc.created_at, pc.updated_at
FROM podcast_channel pc
JOIN (SELECT id FROM user WHERE is_admin ORDER BY updated_at DESC LIMIT 1) u;
-- +goose StatementEnd

-- Every already-downloaded episode becomes "downloaded" for that same backfilled subscriber, so
-- their existing downloads stay visible to them under the new per-subscriber visibility model
-- instead of silently disappearing from their list.
-- +goose StatementBegin
INSERT INTO annotation (user_id, item_id, item_type, downloaded, downloaded_at)
SELECT ps.user_id, pe.id, 'podcast_episode', true, pe.updated_at
FROM podcast_episode pe
JOIN podcast_subscription ps ON ps.channel_id = pe.channel_id
WHERE pe.download_status = 'downloaded'
ON CONFLICT (user_id, item_id, item_type) DO UPDATE SET downloaded = true, downloaded_at = excluded.downloaded_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE podcast_channel DROP COLUMN download_policy;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE podcast_channel DROP COLUMN retention_count;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE podcast_channel DROP COLUMN retention_days;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE podcast_channel DROP COLUMN max_storage_mb;
-- +goose StatementEnd

-- +goose Down
