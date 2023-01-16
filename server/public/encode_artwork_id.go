package public

import (
	"context"
	"errors"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
)

func EncodeArtworkID(artID model.ArtworkID) string {
	token, _ := auth.CreatePublicToken(map[string]any{"id": artID.String()})
	return token
}

func DecodeArtworkID(tokenString string) (model.ArtworkID, error) {
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
	return model.ParseArtworkID(id)
}
