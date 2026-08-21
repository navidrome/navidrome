package persistence

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

// Keeps each multi-row insert under SQLite's bind-variable limit (at most 7 vars per row).
const enqueueChunkSize = 100

// Every insert writes these, in this order; the INSERT..SELECT forms must project them to match.
// DequeueBatch also selects exactly these, to leave the drain's rows free of the trace it never reads.
var enqueueColumns = []string{"item_kind", "item_id", "image_type", "priority", "attempts", "retry_at", "enqueued_at"}

type artworkQueueRepository struct {
	sqlRepository
}

func NewArtworkQueueRepository(ctx context.Context, db dbx.Builder) model.ArtworkQueueRepository {
	r := &artworkQueueRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "artwork_queue"
	return r
}

func (r *artworkQueueRepository) Get(kind model.Kind, id, imageType string) (*model.ArtworkQueueItem, error) {
	var res model.ArtworkQueueItem
	err := r.queryOne(Select("*").From(r.tableName).
		Where(Eq{"item_kind": kind.Prefix(), "item_id": id, "image_type": imageType}), &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Enqueue starts a fresh lifecycle: it resets enqueued_at (so a fresh request does not inherit an old
// row's spent retry budget) and clears trace (so explain does not show a prior failure at attempts 0).
func (r *artworkQueueRepository) Enqueue(items ...model.ArtworkQueueItem) error {
	return r.enqueue(`ON CONFLICT (item_kind, item_id, image_type) DO UPDATE SET
		priority = MAX(priority, excluded.priority), retry_at = excluded.retry_at,
		attempts = 0, enqueued_at = excluded.enqueued_at, trace = '[]'`, items)
}

func (r *artworkQueueRepository) EnqueuePreservingBackoff(items ...model.ArtworkQueueItem) error {
	return r.enqueue(`ON CONFLICT (item_kind, item_id, image_type) DO UPDATE SET
		priority = MAX(priority, excluded.priority)`, items)
}

func (r *artworkQueueRepository) EnqueueStaleAbsent(kind model.Kind, attemptedBefore time.Time) (int64, error) {
	now := time.Now()
	return r.insertIfNotQueued("", `SELECT item_kind, item_id, image_type, ?, 0, ?, ?
		FROM `+itemArtworkTable+` WHERE item_kind = ? AND hash = '' AND attempted_at < ?`,
		model.ArtworkPriorityRecheck, now, now, kind.Prefix(), attemptedBefore)
}

func (r *artworkQueueRepository) EnqueueAllMissing(kind model.Kind, priority int) (int64, error) {
	entityTable, ok := artworkOwnerTables[kind]
	if !ok {
		return 0, fmt.Errorf("artwork queue: no entity table for kind %q", kind.Prefix())
	}
	now := time.Now()
	return r.insertIfNotQueued("", `SELECT ?, id, ?, ?, 0, ?, ?
		FROM `+entityTable+`
		WHERE id NOT IN (SELECT item_id FROM `+itemArtworkTable+` WHERE item_kind = ?)`,
		kind.Prefix(), model.ImageTypePrimary, priority, now, now, kind.Prefix())
}

func (r *artworkQueueRepository) EnqueueIfMissing(items ...model.ArtworkQueueItem) error {
	now := time.Now()
	for chunk := range slices.Chunk(items, enqueueChunkSize) {
		rows := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*4+2)
		for _, it := range chunk {
			rows = append(rows, "(?,?,?,?)")
			args = append(args, it.ItemKind, it.ItemID, cmp.Or(it.ImageType, model.ImageTypePrimary), it.Priority)
		}
		args = append(args, now, now)
		_, err := r.insertIfNotQueued(
			`WITH new_items(item_kind, item_id, image_type, priority) AS (VALUES `+strings.Join(rows, ",")+`) `,
			`SELECT n.item_kind, n.item_id, n.image_type, n.priority, 0, ?, ?
			FROM new_items n
			WHERE NOT EXISTS (
				SELECT 1 FROM `+itemArtworkTable+` ia
				WHERE ia.item_kind = n.item_kind AND ia.item_id = n.item_id AND ia.image_type = n.image_type)`,
			args...)
		if err != nil {
			return err
		}
	}
	return nil
}

// DO NOTHING is deliberate: a recheck must not bump the priority or retry_at of an already-queued item.
const skipIfQueued = ` ON CONFLICT (item_kind, item_id, image_type) DO NOTHING`

// insertIfNotQueued inserts the rows selected by the given SQL, optionally prefixed by a CTE.
func (r *artworkQueueRepository) insertIfNotQueued(with, sql string, args ...any) (int64, error) {
	return r.executeSQL(Expr(with+`INSERT INTO `+r.tableName+
		` (`+strings.Join(enqueueColumns, ", ")+`) `+sql+skipIfQueued, args...))
}

// artworkSourceFilter selects item_artwork rows of a kind; no sources means every source, "" the absent state.
func artworkSourceFilter(kind model.Kind, sources []string) Sqlizer {
	f := And{Eq{"item_kind": kind.Prefix()}}
	if len(sources) > 0 {
		f = append(f, Eq{"source": sources})
	}
	return f
}

func (r *artworkQueueRepository) CountBySource(kind model.Kind, sources []string) (int64, error) {
	var res struct{ Count int64 }
	err := r.queryOne(Select("count(*) as count").From(itemArtworkTable).
		Where(artworkSourceFilter(kind, sources)), &res)
	return res.Count, err
}

func (r *artworkQueueRepository) SourcesInUse(kind model.Kind) ([]string, error) {
	var res []struct{ Source string }
	err := r.queryAll(Select("distinct source").From(itemArtworkTable).
		Where(Eq{"item_kind": kind.Prefix()}), &res)
	if err != nil {
		return nil, err
	}
	return slice.Map(res, func(s struct{ Source string }) string { return s.Source }), nil
}

// EnqueueBySource deliberately leaves item_artwork alone: clearing state in bulk would blank the
// library's artwork until every item is resolved again.
func (r *artworkQueueRepository) EnqueueBySource(kind model.Kind, sources []string, priority int) (int64, error) {
	now := time.Now()
	sel := Select("item_kind", "item_id", "image_type").
		Column(Expr("?", priority)).Column("0").Column(Expr("?", now)).Column(Expr("?", now)).
		From(itemArtworkTable).Where(artworkSourceFilter(kind, sources))
	return r.executeSQL(Insert(r.tableName).Columns(enqueueColumns...).Select(sel).Suffix(skipIfQueued))
}

func (r *artworkQueueRepository) enqueue(conflict string, items []model.ArtworkQueueItem) error {
	now := time.Now()
	for chunk := range slices.Chunk(items, enqueueChunkSize) {
		ins := Insert(r.tableName).Columns(enqueueColumns...)
		for _, it := range chunk {
			ins = ins.Values(it.ItemKind, it.ItemID, cmp.Or(it.ImageType, model.ImageTypePrimary), it.Priority, 0, now, now)
		}
		ins = ins.Suffix(conflict)
		if _, err := r.executeSQL(ins); err != nil {
			return err
		}
	}
	return nil
}

func (r *artworkQueueRepository) DequeueBatch(n int, kinds ...string) ([]model.ArtworkQueueItem, error) {
	sel := Select(enqueueColumns...).From(r.tableName).
		Where(LtOrEq{"retry_at": time.Now()}).
		OrderBy("priority DESC", "enqueued_at ASC").
		Limit(uint64(n))
	if len(kinds) > 0 {
		sel = sel.Where(Eq{"item_kind": kinds})
	}
	var res []model.ArtworkQueueItem
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *artworkQueueRepository) MarkFailedIfUnchanged(kind, id, imageType string, seenRetryAt, retryAt time.Time, trace string) error {
	upd := Update(r.tableName).
		Set("attempts", Expr("attempts + 1")).
		Set("retry_at", retryAt).
		Set("trace", trace).
		Where(Eq{"item_kind": kind, "item_id": id, "image_type": imageType, "retry_at": seenRetryAt})
	_, err := r.executeSQL(upd)
	return err
}

func (r *artworkQueueRepository) DeleteIfUnchanged(kind, id, imageType string, retryAt time.Time) error {
	return r.delete(Eq{"item_kind": kind, "item_id": id, "image_type": imageType, "retry_at": retryAt})
}

func (r *artworkQueueRepository) PurgeDangling() (int64, error) {
	return purgeDangling(r.sqlRepository)
}

// PurgeQueued ignores retry_at: a row still backing off is pending work that would drain later.
func (r *artworkQueueRepository) PurgeQueued(kinds []model.Kind, priorities []int) (int64, error) {
	del := Delete(r.tableName)
	if len(kinds) > 0 {
		del = del.Where(Eq{"item_kind": slice.Map(kinds, func(k model.Kind) string { return k.Prefix() })})
	}
	if len(priorities) > 0 {
		del = del.Where(Eq{"priority": priorities})
	}
	return r.executeSQL(del)
}

func (r *artworkQueueRepository) Count() (int64, error) {
	var res struct{ Count int64 }
	err := r.queryOne(Select("count(*) as count").From(r.tableName), &res)
	return res.Count, err
}

func (r *artworkQueueRepository) CountByKindAndPriority() ([]model.ArtworkQueueStat, error) {
	var res []model.ArtworkQueueStat
	err := r.queryAll(Select("item_kind", "priority", "count(*) as count").From(r.tableName).
		GroupBy("item_kind", "priority").OrderBy("item_kind", "priority desc"), &res)
	return res, err
}

// CountAbsent matches EnqueueStaleAbsent on hash, so the stale count is what a recheck would queue.
func (r *artworkQueueRepository) CountAbsent(kind model.Kind, attemptedBefore time.Time) (model.ArtworkAbsentStat, error) {
	var res model.ArtworkAbsentStat
	err := r.queryOne(Select("count(*) as total").
		Column(Expr("coalesce(sum(attempted_at < ?), 0) as stale", attemptedBefore)).
		From(itemArtworkTable).Where(Eq{"item_kind": kind.Prefix(), "hash": ""}), &res)
	return res, err
}

var _ model.ArtworkQueueRepository = (*artworkQueueRepository)(nil)
