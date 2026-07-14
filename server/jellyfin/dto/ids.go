package dto

import "encoding/hex"

// EncodeID renders a Navidrome id as lowercase hex; Jellyfin clients parse ids as radix-16 (e.g.
// Finamp's queue packing) and crash on Navidrome's base62 nanoids if emitted as-is.
func EncodeID(id string) string {
	if id == "" {
		return ""
	}
	return hex.EncodeToString([]byte(id))
}

// DecodeID reverses EncodeID; non-hex input is returned unchanged, so it's safe on any inbound id.
func DecodeID(id string) string {
	if id == "" {
		return ""
	}
	if b, err := hex.DecodeString(id); err == nil && len(b) > 0 {
		return string(b)
	}
	return id
}
