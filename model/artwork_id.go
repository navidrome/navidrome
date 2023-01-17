package model

import (
	"errors"
	"fmt"
	"strings"
)

type Kind struct {
	prefix string
	name   string
}

func (k Kind) String() string {
	return k.name
}

var (
	KindMediaFileArtwork = Kind{"mf", "media_file"}
	KindArtistArtwork    = Kind{"ar", "artist"}
	KindAlbumArtwork     = Kind{"al", "album"}
	KindPlaylistArtwork  = Kind{"pl", "playlist"}
)

var artworkKindMap = map[string]Kind{
	KindMediaFileArtwork.prefix: KindMediaFileArtwork,
	KindArtistArtwork.prefix:    KindArtistArtwork,
	KindAlbumArtwork.prefix:     KindAlbumArtwork,
	KindPlaylistArtwork.prefix:  KindPlaylistArtwork,
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
	if kind, ok := artworkKindMap[parts[0]]; !ok {
		return ArtworkID{}, errors.New("invalid artwork kind")
	} else {
		return ArtworkID{
			Kind: kind,
			ID:   parts[1],
		}, nil
	}
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

func artworkIDFromArtist(ar Artist) ArtworkID {
	return ArtworkID{
		Kind: KindArtistArtwork,
		ID:   ar.ID,
	}
}
