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
	MediaFileArtwork = Kind{"mf"}
	AlbumArtwork     = Kind{"al"}
)

type ArtworkID struct {
	Kind       Kind
	ID         string
	LastAccess time.Time
}

func (id ArtworkID) String() string {
	return fmt.Sprintf("%s-%s-%x", id.Kind.prefix, id.ID, id.LastAccess.Unix())
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
	if parts[0] != AlbumArtwork.prefix && parts[0] != MediaFileArtwork.prefix {
		return ArtworkID{}, errors.New("invalid artwork kind")
	}
	return ArtworkID{
		Kind:       Kind{parts[0]},
		ID:         parts[1],
		LastAccess: time.Unix(lastUpdate, 0),
	}, nil
}

func artworkIDFromAlbum(al Album) ArtworkID {
	return ArtworkID{
		Kind:       AlbumArtwork,
		ID:         al.ID,
		LastAccess: al.UpdatedAt,
	}
}

func artworkIDFromMediaFile(mf MediaFile) ArtworkID {
	return ArtworkID{
		Kind:       MediaFileArtwork,
		ID:         mf.ID,
		LastAccess: mf.UpdatedAt,
	}
}
