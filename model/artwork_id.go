package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Kind struct{ prefix string }

var (
	KindMediaFileArtwork = Kind{"mf"}
	KindAlbumArtwork     = Kind{"al"}
)

type ArtworkID struct {
	Kind       Kind
	ID         string
	LastUpdate time.Time
}

func (id ArtworkID) String() string {
	s := fmt.Sprintf("%s-%s", id.Kind.prefix, id.ID)
	if id.LastUpdate.Unix() < 0 {
		return s + "-0"
	}
	return fmt.Sprintf("%s-%x", s, id.LastUpdate.Unix())
}

func ParseArtworkID(id string) (ArtworkID, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		return ArtworkID{}, errors.New("invalid artwork id")
	}
	lastUpdate, err := strconv.ParseInt(parts[2], 16, 64)
	if err != nil {
		return ArtworkID{}, err
	}
	if parts[0] != KindAlbumArtwork.prefix && parts[0] != KindMediaFileArtwork.prefix {
		return ArtworkID{}, errors.New("invalid artwork kind")
	}
	return ArtworkID{
		Kind:       Kind{parts[0]},
		ID:         parts[1],
		LastUpdate: time.Unix(lastUpdate, 0),
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
		Kind:       KindAlbumArtwork,
		ID:         al.ID,
		LastUpdate: al.UpdatedAt,
	}
}

func artworkIDFromMediaFile(mf MediaFile) ArtworkID {
	return ArtworkID{
		Kind:       KindMediaFileArtwork,
		ID:         mf.ID,
		LastUpdate: mf.UpdatedAt,
	}
}
