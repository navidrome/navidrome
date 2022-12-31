package model

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

type Kind struct{ prefix string }

var (
	KindMediaFileArtwork = Kind{"mf"}
	KindArtistArtwork    = Kind{"ar"}
	KindAlbumArtwork     = Kind{"al"}
	KindPlaylistArtwork  = Kind{"pl"}
)

var artworkKindList = []string{
	KindMediaFileArtwork.prefix,
	KindArtistArtwork.prefix,
	KindAlbumArtwork.prefix,
	KindPlaylistArtwork.prefix,
}

type ArtworkID struct {
	Kind Kind
	ID   string
}

func (id ArtworkID) String() string {
	if id.ID == "" {
		return ""
	}
	return fmt.Sprintf("%s-%s", id.Kind.prefix, id.ID)
}

func NewArtworkID(kind Kind, id string) ArtworkID {
	return ArtworkID{kind, id}
}

func ParseArtworkID(id string) (ArtworkID, error) {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) != 2 {
		return ArtworkID{}, errors.New("invalid artwork id")
	}
	if !slices.Contains(artworkKindList, parts[0]) {
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

func artworkIDFromPlaylist(pls Playlist) ArtworkID {
	return ArtworkID{
		Kind: KindPlaylistArtwork,
		ID:   pls.ID,
	}
}
