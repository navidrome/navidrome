package jellyfin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
)

// resolveAnnotated finds which repo (album, artist or media file) owns id, mirroring getItem's
// probe order and access control: albums and songs each belong to exactly one library and 404
// when the current user can't access it, so an id can't be used to favorite/rate content outside
// the user's libraries by guessing. Artists span multiple libraries (via library_artist), so —
// same as getItem — there's no single LibraryID to check; access for them relies on list-time
// scoping and the persistence layer's defense-in-depth.
// ok is false, and the response has already been written, when the id doesn't resolve to any
// entity or access is denied; callers must return immediately without writing the annotation.
func (api *Router) resolveAnnotated(w http.ResponseWriter, r *http.Request, id string) (repo model.AnnotatedRepository, ok bool) {
	ctx := r.Context()
	u, _ := request.UserFrom(ctx)
	if al, err := api.ds.Album(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(al.LibraryID) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return nil, false
		}
		return api.ds.Album(ctx), true
	}
	if _, err := api.ds.Artist(ctx).Get(id); err == nil {
		return api.ds.Artist(ctx), true
	}
	if mf, err := api.ds.MediaFile(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(mf.LibraryID) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return nil, false
		}
		return api.ds.MediaFile(ctx), true
	}
	http.Error(w, "Not Found", http.StatusNotFound)
	return nil, false
}

func (api *Router) setFavorite(w http.ResponseWriter, r *http.Request, starred bool) {
	id := chi.URLParam(r, "itemId")
	repo, ok := api.resolveAnnotated(w, r, id)
	if !ok {
		return
	}
	if err := repo.SetStar(starred, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	api.ok(w, r, &dto.UserItemDataDto{IsFavorite: starred, Key: id, ItemId: id})
}

func (api *Router) markFavorite(w http.ResponseWriter, r *http.Request) { api.setFavorite(w, r, true) }
func (api *Router) unmarkFavorite(w http.ResponseWriter, r *http.Request) {
	api.setFavorite(w, r, false)
}

func (api *Router) setItemRating(w http.ResponseWriter, r *http.Request, rating int) {
	id := chi.URLParam(r, "itemId")
	repo, ok := api.resolveAnnotated(w, r, id)
	if !ok {
		return
	}
	if err := repo.SetRating(rating, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d := &dto.UserItemDataDto{Key: id, ItemId: id}
	if rating > 0 {
		jfRating := float64(rating) * 2 // Navidrome 0-5 -> Jellyfin 0-10, mirrors dto.UserData
		d.Rating = &jfRating
	}
	api.ok(w, r, d)
}

func (api *Router) setRating(w http.ResponseWriter, r *http.Request) {
	rating := req.Params(r).IntOr("Rating", 0) / 2 // Jellyfin 0-10 -> Navidrome 0-5
	api.setItemRating(w, r, rating)
}

func (api *Router) removeRating(w http.ResponseWriter, r *http.Request) {
	api.setItemRating(w, r, 0)
}
