package subsonic

import (
	"errors"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

func (api *Router) GetBookmarks(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := api.ds.MediaFile(r.Context())
	bookmarks, err := repo.GetBookmarks()
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Bookmarks = &responses.Bookmarks{}
	response.Bookmarks.Bookmark = slice.Map(bookmarks, func(bmk model.Bookmark) responses.Bookmark {
		return responses.Bookmark{
			Entry:    childFromMediaFile(r.Context(), bmk.Item),
			Position: bmk.Position,
			Username: user.UserName,
			Comment:  bmk.Comment,
			Created:  bmk.CreatedAt,
			Changed:  bmk.UpdatedAt,
		}
	})
	return response, nil
}

func (api *Router) CreateBookmark(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	comment, _ := p.String("comment")
	position := p.Int64Or("position", 0)

	repo := api.ds.MediaFile(r.Context())
	err = repo.AddBookmark(id, comment, position)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) DeleteBookmark(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	repo := api.ds.MediaFile(r.Context())
	err = repo.DeleteBookmark(id)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) GetPlayQueue(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := api.ds.PlayQueue(r.Context())
	pq, err := repo.RetrieveWithMediaFiles(user.ID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}
	if pq == nil || len(pq.Items) == 0 {
		return newResponse(), nil
	}

	response := newResponse()
	var currentID string
	if pq.Current >= 0 && pq.Current < len(pq.Items) {
		currentID = pq.Items[pq.Current].ID
	}
	response.PlayQueue = &responses.PlayQueue{
		Entry:     slice.MapWithArg(pq.Items, r.Context(), childFromMediaFile),
		Current:   currentID,
		Position:  pq.Position,
		Username:  user.UserName,
		Changed:   &pq.UpdatedAt,
		ChangedBy: pq.ChangedBy,
	}
	return response, nil
}

func (api *Router) SavePlayQueue(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids, _ := p.Strings("id")
	currentID, _ := p.String("current")
	position := p.Int64Or("position", 0)

	user, _ := request.UserFrom(r.Context())
	client, _ := request.ClientFrom(r.Context())

	items := slice.Map(ids, func(id string) model.MediaFile {
		return model.MediaFile{ID: id}
	})

	currentIndex := 0
	for i, id := range ids {
		if id == currentID {
			currentIndex = i
			break
		}
	}

	pq := &model.PlayQueue{
		UserID:    user.ID,
		Current:   currentIndex,
		Position:  position,
		ChangedBy: client,
		Items:     items,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	repo := api.ds.PlayQueue(r.Context())
	err := repo.Store(pq)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}
