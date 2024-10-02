package public

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	. "github.com/navidrome/navidrome/utils/gg"
)

func ImageURL(r *http.Request, artID model.ArtworkID, size int) string {
	token := encodeArtworkID(artID)
	uri := path.Join(consts.URLPathPublicImages, token)
	params := url.Values{}
	if size > 0 {
		params.Add("size", strconv.Itoa(size))
	}
	return publicURL(r, uri, params)
}

func encodeArtworkID(artID model.ArtworkID) string {
	token, _ := auth.CreatePublicToken(map[string]any{"id": artID.String()})
	return token
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

func encodeMediafileShare(s model.Share, id string) string {
	claims := map[string]any{"id": id}
	if s.Format != "" {
		claims["f"] = s.Format
	}
	if s.MaxBitRate != 0 {
		claims["b"] = s.MaxBitRate
	}
	token, _ := auth.CreateExpiringPublicToken(V(s.ExpiresAt), claims)
	return token
}
