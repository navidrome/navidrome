package jellyfin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

type annotatedItem struct {
	repo     model.AnnotatedRepository
	resource string // name used in refreshResource events
	ann      model.Annotations
}

// resolveAnnotated finds id's repo and the caller's annotations. Only albums and songs are library-gated
// (artists span libraries; playlists gate in Get). When !ok, the response has already been written.
func (api *Router) resolveAnnotated(w http.ResponseWriter, r *http.Request, id string) (item annotatedItem, ok bool) {
	ctx := r.Context()
	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		api.internalError(w, r, err)
		return annotatedItem{}, false
	}
	u, _ := request.UserFrom(ctx)
	switch e := entity.(type) {
	case *model.Album:
		if u.HasLibraryAccess(e.LibraryID) {
			return annotatedItem{api.ds.Album(), "album", e.Annotations}, true
		}
	case *model.Artist:
		return annotatedItem{api.ds.Artist(), "artist", e.Annotations}, true
	case *model.MediaFile:
		if u.HasLibraryAccess(e.LibraryID) {
			return annotatedItem{api.ds.MediaFile(), "song", e.Annotations}, true
		}
	case *model.Playlist:
		return annotatedItem{api.ds.Playlist(), "playlist", e.Annotations}, true
	}
	// Unknown ids, inaccessible-library items and non-annotatable entities (radios) all read as absent.
	http.Error(w, "Not Found", http.StatusNotFound)
	return annotatedItem{}, false
}

// getUserItemData returns the caller's play/favorite/rating state for a single item. Jellify
// fetches this per item to render played/favourite indicators; resolveItemByID enforces the
// library-access gate.
func (api *Router) getUserItemData(w http.ResponseWriter, r *http.Request) {
	id, ok := itemIDParam(w, r, "itemId")
	if !ok {
		return
	}
	item, ok := api.resolveItemByID(r.Context(), id, nil)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	data := item.UserData
	if data == nil {
		// Items without annotations still return a valid empty UserData.
		data = dto.UserData(model.Annotations{}, id)
	}
	api.ok(w, r, data)
}

// userDataChange holds already-validated writes; a nil field leaves that annotation untouched.
type userDataChange struct {
	rating   *int // Navidrome stars; 0 clears
	favorite *bool
}

// applyUserData writes change for the itemId path param and answers with the item's resulting
// UserData, as Jellyfin does.
func (api *Router) applyUserData(w http.ResponseWriter, r *http.Request, change userDataChange) {
	id, ok := itemIDParam(w, r, "itemId")
	if !ok {
		return
	}
	item, ok := api.resolveAnnotated(w, r, id)
	if !ok {
		return
	}
	ctx := r.Context()
	if change.rating != nil {
		if err := item.repo.SetRating(ctx, *change.rating, id); err != nil {
			api.internalError(w, r, err)
			return
		}
		item.ann.Rating = *change.rating
	}
	if change.favorite != nil {
		if err := item.repo.SetStar(ctx, *change.favorite, id); err != nil {
			api.internalError(w, r, err)
			return
		}
		item.ann.Starred = *change.favorite
	}
	if change.rating != nil || change.favorite != nil {
		api.broker.SendMessage(ctx, (&events.RefreshResource{}).With(item.resource, id))
	}
	api.ok(w, r, dto.UserData(item.ann, id))
}

// applyCallerUserData is applyUserData for routes that may name a userId: only the caller's own
// annotations can be written, so naming another user is refused. Undecodable ids are ignored.
func (api *Router) applyCallerUserData(w http.ResponseWriter, r *http.Request, change userDataChange) {
	raw := chi.URLParam(r, "userId")
	if raw == "" {
		raw = r.URL.Query().Get("userid")
	}
	if target, ok := dto.DecodeID(raw); ok {
		if u, _ := request.UserFrom(r.Context()); target != u.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}
	api.applyUserData(w, r, change)
}

func (api *Router) markFavorite(w http.ResponseWriter, r *http.Request) {
	api.applyUserData(w, r, userDataChange{favorite: new(true)})
}

func (api *Router) unmarkFavorite(w http.ResponseWriter, r *http.Request) {
	api.applyUserData(w, r, userDataChange{favorite: new(false)})
}

// setRating reads Jellyfin's likes flag, or the numeric rating extension older clients send. Unlike
// Jellyfin, a missing or malformed value is a 400 rather than a clear: clearing is DELETE's job.
func (api *Router) setRating(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var stars int
	switch {
	case q.Has("likes"):
		likes, err := strconv.ParseBool(q.Get("likes"))
		if err != nil {
			http.Error(w, "likes must be true or false", http.StatusBadRequest)
			return
		}
		stars = dto.StarsFromLikes(likes)
	case q.Has("rating"):
		rating, err := strconv.ParseFloat(q.Get("rating"), 64)
		s, ok := dto.StarsFromRating(rating)
		if err != nil || !ok {
			http.Error(w, "rating must be a number from 0 to 10", http.StatusBadRequest)
			return
		}
		stars = s
	default:
		http.Error(w, "likes is required", http.StatusBadRequest)
		return
	}
	api.applyCallerUserData(w, r, userDataChange{rating: &stars})
}

func (api *Router) removeRating(w http.ResponseWriter, r *http.Request) {
	api.applyCallerUserData(w, r, userDataChange{rating: new(0)})
}

// updateUserItemData writes Rating, Likes and IsFavorite; omitted or null fields are left untouched.
// Play state is ignored because it only changes through playback reporting.
func (api *Router) updateUserItemData(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeUserDataBody(w, r)
	if !ok {
		return
	}
	change := userDataChange{favorite: body.IsFavorite}
	// Jellyfin applies Likes before Rating, so Rating wins when a body carries both.
	switch {
	case body.Rating != nil:
		stars, ok := dto.StarsFromRating(*body.Rating)
		if !ok {
			http.Error(w, "Rating must be from 0 to 10", http.StatusBadRequest)
			return
		}
		change.rating = &stars
	case body.Likes != nil:
		change.rating = new(dto.StarsFromLikes(*body.Likes))
	}
	api.applyCallerUserData(w, r, change)
}

const maxUserDataBody = 64 << 10

// decodeUserDataBody accepts exactly one JSON object; Decode alone would ignore anything after it.
func decodeUserDataBody(w http.ResponseWriter, r *http.Request) (*dto.UpdateUserItemDataDto, bool) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxUserDataBody))
	var body *dto.UpdateUserItemDataDto
	err := dec.Decode(&body)
	if err == nil && body != nil {
		if _, err = dec.Token(); errors.Is(err, io.EOF) {
			return body, true
		}
	}
	http.Error(w, "Bad Request", http.StatusBadRequest)
	return nil, false
}
