package persistence

import (
	"context"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type playQueueRepository struct {
	sqlRepository
	sqlRestful
}

func NewPlayQueueRepository(ctx context.Context, o orm.Ormer) model.PlayQueueRepository {
	r := &playQueueRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "playqueue"
	return r
}

type playQueue struct {
	ID        string `orm:"column(id)"`
	UserID    string `orm:"column(user_id)"`
	Comment   string
	Current   string
	Position  float32
	ChangedBy string
	Items     string
	CreatedAt time.Time
	UpdatedAt time.Time
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
		Comment:   q.Comment,
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
		Comment:   pq.Comment,
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
	q.Items = r.loadTracks(&q)
	return q
}

func (r *playQueueRepository) loadTracks(p *model.PlayQueue) model.MediaFiles {
	if len(p.Items) == 0 {
		return nil
	}

	// Collect all ids
	ids := make([]string, len(p.Items))
	for i, t := range p.Items {
		ids[i] = t.ID
	}

	// Break the list in chunks, up to 50 items, to avoid hitting SQLITE_MAX_FUNCTION_ARG limit
	const chunkSize = 50
	var chunks [][]string
	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize
		if end > len(ids) {
			end = len(ids)
		}

		chunks = append(chunks, ids[i:end])
	}

	// Query each chunk of media_file ids and store results in a map
	mfRepo := NewMediaFileRepository(r.ctx, r.ormer)
	trackMap := map[string]model.MediaFile{}
	for i := range chunks {
		idsFilter := Eq{"id": chunks[i]}
		tracks, err := mfRepo.GetAll(model.QueryOptions{Filters: idsFilter})
		if err != nil {
			u := loggedUser(r.ctx)
			log.Error(r.ctx, "Could not load playqueue's tracks", "user", u.UserName, err)
		}
		for _, t := range tracks {
			trackMap[t.ID] = t
		}
	}

	// Create a new list of tracks with the same order as the original
	newTracks := make(model.MediaFiles, len(p.Items))
	for i, t := range p.Items {
		newTracks[i] = trackMap[t.ID]
	}
	return newTracks
}

func (r *playQueueRepository) clearPlayQueue(userId string) error {
	return r.delete(Eq{"user_id": userId})
}

var _ model.PlayQueueRepository = (*playQueueRepository)(nil)
