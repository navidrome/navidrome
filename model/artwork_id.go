package model

import (
	"errors"
	"fmt"
	"strings"
)

type Kind struct{ prefix string }

var (
	KindMediaFileArtwork = Kind{"mf"}
	KindAlbumArtwork     = Kind{"al"}
)

type ArtworkID struct {
	Kind Kind
	ID   string
}

func (id ArtworkID) String() string {
	return fmt.Sprintf("%s-%s", id.Kind.prefix, id.ID)
}

func ParseArtworkID(id string) (ArtworkID, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return ArtworkID{}, errors.New("invalid artwork id")
	}
	if parts[0] != KindAlbumArtwork.prefix && parts[0] != KindMediaFileArtwork.prefix {
		return ArtworkID{}, errors.New("invalid artwork kind")
	}
	return ArtworkID{
		Kind: Kind{parts[0]},
		ID:   parts[1],
	}, nil
}

func MustParseArtworkID(id string) ArtworkID {
	artID, err := ParseArtworkID(id)
	if err != nil {
		panic(artID)
	}
	return artID
}

func artworkIDFromAlbum(al Album) ArtworkID {
	return ArtworkID{
		Kind: KindAlbumArtwork,
		ID:   al.ID,
	}
}

func artworkIDFromMediaFile(mf MediaFile) ArtworkID {
	return ArtworkID{
		Kind: KindMediaFileArtwork,
		ID:   mf.ID,
	}
}
