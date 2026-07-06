package dto

import "encoding/hex"

// EncodeID renders a Navidrome id as lowercase hex so Jellyfin clients (which parse ids as
// radix-16, e.g. Finamp's queue packing) accept it. Navidrome's base62 nanoids aren't valid
// hex and crash such clients if emitted as-is.
func EncodeID(id string) string {
	if id == "" {
		return ""
	}
	return hex.EncodeToString([]byte(id))
}

// DecodeID reverses EncodeID. Non-hex input (a raw id, or one that's already been decoded) is
// returned unchanged, so it's safe to call on any inbound id even though every id Navidrome
// itself emits is hex-encoded.
func DecodeID(id string) string {
	if id == "" {
		return ""
	}
	if b, err := hex.DecodeString(id); err == nil && len(b) > 0 {
		return string(b)
	}
	return id
}
