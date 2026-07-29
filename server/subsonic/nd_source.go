package subsonic

// validCirqueSource reports whether source is a recognized nd_source device-type
// value. Empty string is valid too - standard Subsonic clients never send
// nd_source at all, and that must not be treated as invalid.
func validCirqueSource(source string) bool {
	switch source {
	case "", "android_phone", "android_tablet", "android_tv", "android_auto", "windows_desktop":
		return true
	default:
		return false
	}
}
