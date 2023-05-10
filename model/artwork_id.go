package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
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
	Kind       Kind
	ID         string
	LastUpdate time.Time
}

func (id ArtworkID) String() string {
	if id.ID == "" {
		return ""
	}
	s := fmt.Sprintf("%s-%s", id.Kind.prefix, id.ID)
	if lu := id.LastUpdate.Unix(); lu > 0 {
		return fmt.Sprintf("%s_%x", s, lu)
	}
	return s + "_0"
}

func NewArtworkID(kind Kind, id string, lastUpdate *time.Time) ArtworkID {
	artID := ArtworkID{kind, id, time.Time{}}
	if lastUpdate != nil {
		artID.LastUpdate = *lastUpdate
	}
	return artID
}

func ParseArtworkID(id string) (ArtworkID, error) {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) != 2 {
		return ArtworkID{}, errors.New("invalid artwork id")
	}
	kind, ok := artworkKindMap[parts[0]]
	if !ok {
		return ArtworkID{}, errors.New("invalid artwork kind")
	}
	parsedID := ArtworkID{
		Kind: kind,
		ID:   parts[1],
	}
	parts = strings.SplitN(parts[1], "_", 2)
	if len(parts) == 2 {
		if parts[1] != "0" {
			lastUpdate, err := strconv.ParseInt(parts[1], 16, 64)
			if err != nil {
				return ArtworkID{}, err
			}
			parsedID.LastUpdate = time.Unix(lastUpdate, 0)
		}
		parsedID.ID = parts[0]
	}
	return parsedID, nil
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

func artworkIDFromPlaylist(pls Playlist) ArtworkID {
	return ArtworkID{
		Kind:       KindPlaylistArtwork,
		ID:         pls.ID,
		LastUpdate: pls.UpdatedAt,
	}
}

func artworkIDFromArtist(ar Artist) ArtworkID {
	return ArtworkID{
		Kind: KindArtistArtwork,
		ID:   ar.ID,
	}
}
