package subsonic

import (
	"net/http"
	"time"

	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type BookmarksController struct {
	ds model.DataStore
}

func NewBookmarksController(ds model.DataStore) *BookmarksController {
	return &BookmarksController{ds: ds}
}

func (c *BookmarksController) GetBookmarks(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := c.ds.PlayQueue(r.Context())
	bmks, err := repo.GetBookmarks(user.ID)
	if err != nil {
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.Bookmarks = &responses.Bookmarks{}
	for _, bmk := range bmks {
		b := responses.Bookmark{
			Entry:    []responses.Child{ChildFromMediaFile(r.Context(), bmk.Item)},
			Position: bmk.Position,
			Username: user.UserName,
			Comment:  bmk.Comment,
			Created:  bmk.CreatedAt,
			Changed:  bmk.UpdatedAt,
		}
		response.Bookmarks.Bookmark = append(response.Bookmarks.Bookmark, b)
	}
	return response, nil
}

func (c *BookmarksController) CreateBookmark(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}

	comment := utils.ParamString(r, "comment")
	position := utils.ParamInt64(r, "position", 0)

	user, _ := request.UserFrom(r.Context())

	repo := c.ds.PlayQueue(r.Context())
	err = repo.AddBookmark(user.ID, id, comment, position)
	if err != nil {
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}
	return NewResponse(), nil
}

func (c *BookmarksController) DeleteBookmark(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}

	user, _ := request.UserFrom(r.Context())

	repo := c.ds.PlayQueue(r.Context())
	err = repo.DeleteBookmark(user.ID, id)
	if err != nil {
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}
	return NewResponse(), nil
}

func (c *BookmarksController) GetPlayQueue(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := c.ds.PlayQueue(r.Context())
	pq, err := repo.Retrieve(user.ID)
	if err != nil {
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	response := NewResponse()
	response.PlayQueue = &responses.PlayQueue{
		Entry:     ChildrenFromMediaFiles(r.Context(), pq.Items),
		Current:   pq.Current,
		Position:  pq.Position,
		Username:  user.UserName,
		Changed:   &pq.UpdatedAt,
		ChangedBy: pq.ChangedBy,
	}
	return response, nil
}

func (c *BookmarksController) SavePlayQueue(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ids, err := RequiredParamStrings(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}

	current := utils.ParamString(r, "current")
	position := utils.ParamInt64(r, "position", 0)

	user, _ := request.UserFrom(r.Context())
	client, _ := request.ClientFrom(r.Context())

	var items model.MediaFiles
	for _, id := range ids {
		items = append(items, model.MediaFile{ID: id})
	}

	pq := &model.PlayQueue{
		UserID:    user.ID,
		Current:   current,
		Position:  position,
		ChangedBy: client,
		Items:     items,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	repo := c.ds.PlayQueue(r.Context())
	err = repo.Store(pq)
	if err != nil {
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}
	return NewResponse(), nil
}
