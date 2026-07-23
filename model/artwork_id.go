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

// Prefix is the short token used in artwork ids and the item_artwork.item_kind column.
func (k Kind) Prefix() string {
	return k.prefix
}

var (
	KindMediaFileArtwork = Kind{"mf", "media_file"}
	KindArtistArtwork    = Kind{"ar", "artist"}
	KindAlbumArtwork     = Kind{"al", "album"}
	KindPlaylistArtwork  = Kind{"pl", "playlist"}
	KindDiscArtwork      = Kind{"dc", "disc"}
	KindRadioArtwork     = Kind{"ra", "radio"}
)

var artworkKindMap = map[string]Kind{
	KindMediaFileArtwork.prefix: KindMediaFileArtwork,
	KindArtistArtwork.prefix:    KindArtistArtwork,
	KindAlbumArtwork.prefix:     KindAlbumArtwork,
	KindPlaylistArtwork.prefix:  KindPlaylistArtwork,
	KindDiscArtwork.prefix:      KindDiscArtwork,
	KindRadioArtwork.prefix:     KindRadioArtwork,
}

type ArtworkID struct {
	Kind       Kind
	ID         string
	Hash       string    // content-hash suffix; "" = unknown/none
	LastUpdate time.Time // legacy: populated only when parsing old _<hexTimestamp> tokens
}

func (id ArtworkID) String() string {
	if id.ID == "" {
		return ""
	}
	s := fmt.Sprintf("%s-%s", id.Kind.prefix, id.ID)
	if id.Hash != "" {
		return s + "_" + id.Hash
	}
	return s
}

func NewArtworkID(kind Kind, id string, lastUpdate *time.Time) ArtworkID {
	artID := ArtworkID{Kind: kind, ID: id}
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
		parsedID.ID = parts[0]
		suffix := parts[1]
		switch {
		// Hash detection must come first: a 16-hex value with the high bit set overflows int64.
		case isImageHash(suffix):
			parsedID.Hash = suffix
		case suffix != "0":
			if lastUpdate, err := strconv.ParseInt(suffix, 16, 64); err == nil {
				parsedID.LastUpdate = time.Unix(lastUpdate, 0)
			}
		}
	}
	return parsedID, nil
}

// isImageHash reports whether s is a 16-char lowercase-hex XXH3-64 content hash.
func isImageHash(s string) bool {
	if len(s) != 16 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func MustParseArtworkID(id string) ArtworkID {
	artID, err := ParseArtworkID(id)
	if err != nil {
		panic(artID)
	}
	return artID
}

func DiscArtworkID(albumID string, discNumber int) string {
	return fmt.Sprintf("%s:%d", albumID, discNumber)
}

func ParseDiscArtworkID(id string) (albumID string, discNumber int, err error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", 0, errors.New("invalid disc artwork id")
	}
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid disc number in artwork id: %w", err)
	}
	return parts[0], num, nil
}

func artworkIDFromAlbum(al Album) ArtworkID {
	return ArtworkID{Kind: KindAlbumArtwork, ID: al.ID, Hash: al.ImageHash}
}

func artworkIDFromMediaFile(mf MediaFile) ArtworkID {
	return ArtworkID{Kind: KindMediaFileArtwork, ID: mf.ID, Hash: mf.ImageHash}
}

func artworkIDFromPlaylist(pls Playlist) ArtworkID {
	return ArtworkID{Kind: KindPlaylistArtwork, ID: pls.ID, Hash: pls.ImageHash}
}

func artworkIDFromArtist(ar Artist) ArtworkID {
	return ArtworkID{Kind: KindArtistArtwork, ID: ar.ID, Hash: ar.ImageHash}
}

func artworkIDFromRadio(r Radio) ArtworkID {
	return ArtworkID{Kind: KindRadioArtwork, ID: r.ID, Hash: r.ImageHash}
}
