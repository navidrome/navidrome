package dto

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/model/id"
)

// guidLen is the length of a Jellyfin GUID on the wire: 16 bytes as lowercase hex, no dashes
// (what Guid.ToString("N") produces).
const guidLen = 32

// reservedPrefix marks GUIDs standing in for ids that aren't 128-bit values: 12 zero bytes, a
// non-zero kind tag, then a 24-bit payload. The tag is never zero because Jellyfin serializes
// the all-zero GUID as null.
const reservedPrefix = "000000000000000000000000"

const (
	kindLibrary         = "01"
	kindPlaylistsFolder = "02"
	kindPlaylistEntry   = "03"
)

const (
	// PlaylistsFolderID is the internal id of the synthetic "playlists library" folder.
	PlaylistsFolderID = "playlists"
	// PlaylistsFolderGUID is its wire form.
	PlaylistsFolderGUID = reservedPrefix + kindPlaylistsFolder + "000000"
)

// EncodeID renders a canonical Navidrome id as a Jellyfin GUID. Anything that isn't one encodes
// to "" rather than to a shape clients can't parse.
func EncodeID(ndID string) string {
	b, err := id.Decode(ndID)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// EncodeLibraryID renders a library's integer id in the reserved GUID space.
func EncodeLibraryID(libID int) string {
	return encodeReserved(kindLibrary, uint64(libID))
}

// EncodePlaylistEntryID renders a playlist entry's position (model.PlaylistTrack.ID, an integer
// column) in the reserved GUID space. Clients echo it back to remove one occurrence of a song.
func EncodePlaylistEntryID(entryID string) string {
	n, err := strconv.ParseUint(entryID, 10, 24)
	if err != nil {
		return ""
	}
	return encodeReserved(kindPlaylistEntry, n)
}

func encodeReserved(kind string, payload uint64) string {
	return reservedPrefix + kind + fmt.Sprintf("%06x", payload)
}

// DecodeID maps an inbound GUID back to the identifier the rest of the API uses: a canonical id,
// a decimal library id, or PlaylistsFolderID. Dashed and uppercase forms are accepted, as
// Jellyfin's Guid.Parse accepts them. Malformed input yields "", which callers surface as a 404.
func DecodeID(guid string) string {
	guid = strings.ToLower(strings.ReplaceAll(guid, "-", ""))
	if len(guid) != guidLen {
		return ""
	}
	b, err := hex.DecodeString(guid)
	if err != nil {
		return ""
	}
	if rest, ok := strings.CutPrefix(guid, reservedPrefix); ok {
		return decodeReserved(rest)
	}
	return id.Encode([16]byte(b))
}

func decodeReserved(rest string) string {
	payload, err := strconv.ParseUint(rest[2:], 16, 32)
	if err != nil {
		return ""
	}
	switch rest[:2] {
	case kindLibrary, kindPlaylistEntry:
		return strconv.FormatUint(payload, 10)
	case kindPlaylistsFolder:
		return PlaylistsFolderID
	}
	return ""
}
