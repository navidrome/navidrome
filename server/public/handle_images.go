package public

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
)

func (pub *Router) handleImages(w http.ResponseWriter, r *http.Request) {
	// If context is already canceled, discard request without further processing
	if r.Context().Err() != nil {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	p := req.Params(r)
	id, _ := p.String(":id")
	if id == "" {
		log.Warn(r, "No id provided")
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	artId, err := decodeArtworkID(id)
	if err != nil {
		log.Error(r, "Error decoding artwork id", "id", id, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	size := p.IntOr("size", 0)
	square := p.BoolOr("square", false)

	imgReader, lastUpdate, err := pub.artwork.Get(ctx, artId, size, square)
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, model.ErrNotFound):
		log.Warn(r, "Couldn't find coverArt", "id", id, err)
		http.Error(w, "Artwork not found", http.StatusNotFound)
		return
	case errors.Is(err, artwork.ErrUnavailable):
		log.Debug(r, "Item does not have artwork", "id", id, err)
		http.Error(w, "Artwork not found", http.StatusNotFound)
		return
	case err != nil:
		log.Error(r, "Error retrieving coverArt", "id", id, err)
		http.Error(w, "Error retrieving coverArt", http.StatusInternalServerError)
		return
	}

	defer imgReader.Close()
	w.Header().Set("Cache-Control", "public, max-age=315360000")
	w.Header().Set("Last-Modified", lastUpdate.Format(time.RFC1123))
	cnt, err := io.Copy(w, imgReader)
	if err != nil {
		log.Warn(ctx, "Error sending image", "count", cnt, err)
	}
}

func decodeArtworkID(tokenString string) (model.ArtworkID, error) {
	token, err := auth.TokenAuth.Decode(tokenString)
	if err != nil {
		return model.ArtworkID{}, err
	}
	if token == nil {
		return model.ArtworkID{}, errors.New("unauthorized")
	}
	err = jwt.Validate(token, jwt.WithRequiredClaim("id"))
	if err != nil {
		return model.ArtworkID{}, err
	}
	claims, err := token.AsMap(context.Background())
	if err != nil {
		return model.ArtworkID{}, err
	}
	id, ok := claims["id"].(string)
	if !ok {
		return model.ArtworkID{}, errors.New("invalid id type")
	}
	artID, err := model.ParseArtworkID(id)
	if err == nil {
		return artID, nil
	}
	// Try to default to mediafile artworkId (if used with a mediafileShare token)
	return model.ParseArtworkID("mf-" + id)
}
