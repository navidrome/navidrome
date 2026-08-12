package dto

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/model/id"
)

// guidLen is the length of a Jellyfin GUID on the wire: 16 bytes as lowercase hex, no dashes
// (what Guid.ToString("N") produces).
const guidLen = 32

// Reserved GUIDs stand in for ids that aren't 128-bit values: 12 zero bytes, a non-zero kind tag,
// then a 24-bit payload. The tag is never zero because Jellyfin serializes the all-zero GUID as null.
const (
	kindIdx    = 12
	maxPayload = 1<<24 - 1
)

const (
	kindLibrary byte = iota + 1
	kindPlaylistsFolder
	kindPlaylistEntry
)

var zeroPrefix [kindIdx]byte

// PlaylistsFolderID is the internal id of the synthetic "playlists library" folder. It can't be
// mistaken for a real id, which is always 22-char base62.
const PlaylistsFolderID = "playlists"

// PlaylistsFolderGUID is the wire form of PlaylistsFolderID.
var PlaylistsFolderGUID = encodeReserved(kindPlaylistsFolder, 0)

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
	return encodeReserved(kindLibrary, libID)
}

// EncodePlaylistEntryID renders a playlist entry's position (model.PlaylistTrack.ID, an integer
// column) in the reserved GUID space. Clients echo it back to remove one occurrence of a song.
func EncodePlaylistEntryID(entryID string) string {
	n, err := strconv.Atoi(entryID)
	if err != nil {
		return ""
	}
	return encodeReserved(kindPlaylistEntry, n)
}

func encodeReserved(kind byte, payload int) string {
	if payload < 0 || payload > maxPayload {
		return ""
	}
	var b [16]byte
	b[kindIdx] = kind
	b[13], b[14], b[15] = byte(payload>>16), byte(payload>>8), byte(payload)
	return hex.EncodeToString(b[:])
}

// DecodeID maps an inbound GUID back to the identifier the rest of the API uses: a canonical id,
// a decimal library id, or PlaylistsFolderID. Dashed and uppercase forms are accepted, as
// Jellyfin's Guid.Parse accepts them. Malformed input yields "", which callers surface as a 404.
// Playlist entries decode through DecodePlaylistEntryID instead, so a position can't reach a
// caller expecting an entity id.
func DecodeID(guid string) string {
	b, ok := decodeGUID(guid)
	if !ok {
		return ""
	}
	kind, payload, reserved := reservedFields(b)
	if !reserved {
		return id.Encode(b)
	}
	switch kind {
	case kindLibrary:
		return strconv.Itoa(payload)
	case kindPlaylistsFolder:
		return PlaylistsFolderID
	}
	return ""
}

// DecodePlaylistEntryID decodes a playlist entry GUID to its position. It rejects every other kind,
// so an entity id can't be taken for a playlist_tracks row.
func DecodePlaylistEntryID(guid string) (string, bool) {
	b, ok := decodeGUID(guid)
	if !ok {
		return "", false
	}
	if kind, payload, reserved := reservedFields(b); reserved && kind == kindPlaylistEntry {
		return strconv.Itoa(payload), true
	}
	return "", false
}

func decodeGUID(guid string) ([16]byte, bool) {
	guid = strings.ToLower(strings.ReplaceAll(guid, "-", ""))
	if len(guid) != guidLen {
		return [16]byte{}, false
	}
	bs, err := hex.DecodeString(guid)
	if err != nil {
		return [16]byte{}, false
	}
	return [16]byte(bs), true
}

// reservedFields reports the kind tag and payload of a reserved GUID; reserved is false for the
// entity GUIDs that make up almost all traffic.
func reservedFields(b [16]byte) (kind byte, payload int, reserved bool) {
	if !bytes.Equal(b[:kindIdx], zeroPrefix[:]) {
		return 0, 0, false
	}
	return b[kindIdx], int(b[13])<<16 | int(b[14])<<8 | int(b[15]), true
}
