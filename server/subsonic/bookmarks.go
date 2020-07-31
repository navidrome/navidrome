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
		Position:  int64(pq.Position),
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
	position := utils.ParamInt(r, "position", 0)

	user, _ := request.UserFrom(r.Context())
	client, _ := request.ClientFrom(r.Context())

	var items model.MediaFiles
	for _, id := range ids {
		items = append(items, model.MediaFile{ID: id})
	}

	pq := &model.PlayQueue{
		UserID:    user.ID,
		Current:   current,
		Position:  float32(position),
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
