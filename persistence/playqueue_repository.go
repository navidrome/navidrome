package persistence

import (
	"context"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
)

type playQueueRepository struct {
	sqlRepository
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
	Position  int64
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

func (r *playQueueRepository) AddBookmark(userId, id, comment string, position int64) error {
	u := loggedUser(r.ctx)
	client, _ := request.ClientFrom(r.ctx)
	bm := &playQueue{
		UserID:    userId,
		Comment:   comment,
		Current:   id,
		Position:  position,
		ChangedBy: client,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	sel := r.newSelect().Column("id").Where(And{
		Eq{"user_id": userId},
		Eq{"items": ""},
		Eq{"current": id},
	})
	var prev model.PlayQueue
	err := r.queryOne(sel, &prev)
	if err != nil && err != model.ErrNotFound {
		log.Error(r.ctx, "Error retrieving previous bookmark", "user", u.UserName, err, "mediaFileId", id, err)
		return err
	}

	if !prev.CreatedAt.IsZero() {
		bm.CreatedAt = prev.CreatedAt
	}

	_, err = r.put(prev.ID, bm)
	if err != nil {
		log.Error(r.ctx, "Error saving bookmark", "user", u.UserName, err, "mediaFileId", id, err)
		return err
	}
	return nil
}

func (r *playQueueRepository) GetBookmarks(userId string) (model.Bookmarks, error) {
	u := loggedUser(r.ctx)
	sel := r.newSelect().Column("*").Where(And{Eq{"user_id": userId}, Eq{"items": ""}})
	var pqs model.PlayQueues
	err := r.queryAll(sel, &pqs)
	if err != nil {
		log.Error(r.ctx, "Error retrieving bookmarks", "user", u.UserName, err)
		return nil, err
	}
	bms := make(model.Bookmarks, len(pqs))
	for i := range pqs {
		items := r.loadTracks(model.MediaFiles{{ID: pqs[i].Current}})
		bms[i].Item = items[0]
		bms[i].Comment = pqs[i].Comment
		bms[i].Position = int64(pqs[i].Position)
		bms[i].CreatedAt = pqs[i].CreatedAt
		bms[i].UpdatedAt = pqs[i].UpdatedAt
	}
	return bms, nil
}

func (r *playQueueRepository) DeleteBookmark(userId, id string) error {
	return r.delete(And{
		Eq{"user_id": userId},
		Eq{"items": ""},
		Eq{"current": id},
	})
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
