package persistence

import (
	"context"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

type playQueueRepository struct {
	sqlRepository
}

func NewPlayQueueRepository(ctx context.Context, o orm.QueryExecutor) model.PlayQueueRepository {
	r := &playQueueRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "playqueue"
	return r
}

type playQueue struct {
	ID        string    `structs:"id"       orm:"column(id)"`
	UserID    string    `structs:"user_id"  orm:"column(user_id)"`
	Current   string    `structs:"current"`
	Position  int64     `structs:"position"`
	ChangedBy string    `structs:"changed_by"`
	Items     string    `structs:"items"`
	CreatedAt time.Time `structs:"created_at"`
	UpdatedAt time.Time `structs:"updated_at"`
}

func (r *playQueueRepository) Store(q *model.PlayQueue) error {
	u := loggedUser(r.ctx)
	err := r.clearPlayQueue(q.UserID)
	if err != nil {
		log.Error(r.ctx, "Error deleting previous playqueue", "user", u.UserName, err)
		return err
	}
	pq := r.fromModel(q)
	if pq.ID == "" {
		pq.CreatedAt = time.Now()
	}
	pq.UpdatedAt = time.Now()
	_, err = r.put(pq.ID, pq)
	if err != nil {
		log.Error(r.ctx, "Error saving playqueue", "user", u.UserName, err)
		return err
	}
	return nil
}

func (r *playQueueRepository) Retrieve(userId string) (*model.PlayQueue, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"user_id": userId})
	var res playQueue
	err := r.queryOne(sel, &res)
	pls := r.toModel(&res)
	return &pls, err
}

func (r *playQueueRepository) fromModel(q *model.PlayQueue) playQueue {
	pq := playQueue{
		ID:        q.ID,
		UserID:    q.UserID,
		Current:   q.Current,
		Position:  q.Position,
		ChangedBy: q.ChangedBy,
		CreatedAt: q.CreatedAt,
		UpdatedAt: q.UpdatedAt,
	}
	var itemIDs []string
	for _, t := range q.Items {
		itemIDs = append(itemIDs, t.ID)
	}
	pq.Items = strings.Join(itemIDs, ",")
	return pq
}

func (r *playQueueRepository) toModel(pq *playQueue) model.PlayQueue {
	q := model.PlayQueue{
		ID:        pq.ID,
		UserID:    pq.UserID,
		Current:   pq.Current,
		Position:  pq.Position,
		ChangedBy: pq.ChangedBy,
		CreatedAt: pq.CreatedAt,
		UpdatedAt: pq.UpdatedAt,
	}
	if strings.TrimSpace(pq.Items) != "" {
		tracks := strings.Split(pq.Items, ",")
		for _, t := range tracks {
			q.Items = append(q.Items, model.MediaFile{ID: t})
		}
	}
	q.Items = r.loadTracks(q.Items)
	return q
}

func (r *playQueueRepository) loadTracks(tracks model.MediaFiles) model.MediaFiles {
	if len(tracks) == 0 {
		return nil
	}

	// Collect all ids
	ids := make([]string, len(tracks))
	for i, t := range tracks {
		ids[i] = t.ID
	}

	// Break the list in chunks, up to 50 items, to avoid hitting SQLITE_MAX_FUNCTION_ARG limit
	chunks := slice.BreakUp(ids, 50)

	// Query each chunk of media_file ids and store results in a map
	mfRepo := NewMediaFileRepository(r.ctx, r.ormer)
	trackMap := map[string]model.MediaFile{}
	for i := range chunks {
		idsFilter := Eq{"media_file.id": chunks[i]}
		tracks, err := mfRepo.GetAll(model.QueryOptions{Filters: idsFilter})
		if err != nil {
			u := loggedUser(r.ctx)
			log.Error(r.ctx, "Could not load playqueue/bookmark's tracks", "user", u.UserName, err)
		}
		for _, t := range tracks {
			trackMap[t.ID] = t
		}
	}

	// Create a new list of tracks with the same order as the original
	newTracks := make(model.MediaFiles, len(tracks))
	for i, t := range tracks {
		newTracks[i] = trackMap[t.ID]
	}
	return newTracks
}

func (r *playQueueRepository) clearPlayQueue(userId string) error {
	return r.delete(Eq{"user_id": userId})
}

var _ model.PlayQueueRepository = (*playQueueRepository)(nil)
