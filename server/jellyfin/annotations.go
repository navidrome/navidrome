package jellyfin

import (
	"errors"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// resolveAnnotated finds which annotated repo owns id. Albums and songs 404 when the user can't
// access their library; artists span libraries (library_artist), so have no single LibraryID to
// gate on and rely on list-time scoping. PlaylistRepository.Get enforces playlist visibility.
// When ok is false the response has already been written, so callers must return without writing
// the annotation.
func (api *Router) resolveAnnotated(w http.ResponseWriter, r *http.Request, id string) (repo model.AnnotatedRepository, ok bool) {
	ctx := r.Context()
	u, _ := request.UserFrom(ctx)
	if al, err := api.ds.Album(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(al.LibraryID) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return nil, false
		}
		return api.ds.Album(ctx), true
	} else if !errors.Is(err, model.ErrNotFound) {
		api.internalError(w, r, err)
		return nil, false
	}
	if _, err := api.ds.Artist(ctx).Get(id); err == nil {
		return api.ds.Artist(ctx), true
	} else if !errors.Is(err, model.ErrNotFound) {
		api.internalError(w, r, err)
		return nil, false
	}
	if mf, err := api.ds.MediaFile(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(mf.LibraryID) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return nil, false
		}
		return api.ds.MediaFile(ctx), true
	} else if !errors.Is(err, model.ErrNotFound) {
		api.internalError(w, r, err)
		return nil, false
	}
	playlistRepo := api.ds.Playlist(ctx)
	if _, err := playlistRepo.Get(id); err == nil {
		return playlistRepo, true
	} else if !errors.Is(err, model.ErrNotFound) {
		api.internalError(w, r, err)
		return nil, false
	}
	http.Error(w, "Not Found", http.StatusNotFound)
	return nil, false
}

// getUserItemData returns the caller's play/favorite/rating state for a single item. Jellify
// fetches this per item to render played/favourite indicators; resolveItemByID enforces the
// library-access gate.
func (api *Router) getUserItemData(w http.ResponseWriter, r *http.Request) {
	id := api.resolveItemID(r.Context(), dto.DecodeID(chi.URLParam(r, "itemId")))
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

func (api *Router) setFavorite(w http.ResponseWriter, r *http.Request, starred bool) {
	id := api.resolveItemID(r.Context(), dto.DecodeID(chi.URLParam(r, "itemId")))
	repo, ok := api.resolveAnnotated(w, r, id)
	if !ok {
		return
	}
	if err := repo.SetStar(starred, id); err != nil {
		api.internalError(w, r, err)
		return
	}
	encodedID := dto.EncodeID(id)
	api.ok(w, r, &dto.UserItemDataDto{IsFavorite: starred, Key: encodedID, ItemId: encodedID})
}

func (api *Router) markFavorite(w http.ResponseWriter, r *http.Request) { api.setFavorite(w, r, true) }
func (api *Router) unmarkFavorite(w http.ResponseWriter, r *http.Request) {
	api.setFavorite(w, r, false)
}

func (api *Router) setItemRating(w http.ResponseWriter, r *http.Request, rating int) {
	id := api.resolveItemID(r.Context(), dto.DecodeID(chi.URLParam(r, "itemId")))
	repo, ok := api.resolveAnnotated(w, r, id)
	if !ok {
		return
	}
	if err := repo.SetRating(rating, id); err != nil {
		api.internalError(w, r, err)
		return
	}
	encodedID := dto.EncodeID(id)
	d := &dto.UserItemDataDto{Key: encodedID, ItemId: encodedID}
	if rating > 0 {
		jfRating := float64(rating) * 2 // Navidrome 0-5 -> Jellyfin 0-10, mirrors dto.UserData
		d.Rating = &jfRating
	}
	api.ok(w, r, d)
}

// setRating maps Jellyfin's 0-10 rating (a nullable double, so fractional values are valid) to
// Navidrome's 0-5 stars. A nonzero rating floors at one star: rounding to 0 would clear it, since
// SetRating(0) is the delete path.
func (api *Router) setRating(w http.ResponseWriter, r *http.Request) {
	jfRating := req.Params(r).Float64Or("rating", 0)
	jfRating = min(max(jfRating, 0), 10) // clamp: a client sending e.g. Rating=100 must not write an out-of-domain rating
	rating := int(math.Round(jfRating / 2))
	if jfRating > 0 {
		rating = max(rating, 1)
	}
	api.setItemRating(w, r, rating)
}

func (api *Router) removeRating(w http.ResponseWriter, r *http.Request) {
	api.setItemRating(w, r, 0)
}
