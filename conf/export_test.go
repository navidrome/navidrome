package conf

import "fmt"

func ResetConf() {
	current.Store(&Options{})
}

var SetViperDefaults = setViperDefaults

var ParseLanguages = parseLanguages

var ValidateURL = validateURL

var NormalizeSearchBackend = normalizeSearchBackend

var ToPascalCase = toPascalCase

var ValidateByteSize = validateByteSize

var ValidatePlaylistsPath = (*Options).validatePlaylistsPath

func SetRuntimeInfoForTest(goos string, euid int) func() {
	oldGOOS := currentGOOS
	oldEUID := getEUID
	currentGOOS = func() string { return goos }
	getEUID = func() int { return euid }
	return func() {
		currentGOOS = oldGOOS
		getEUID = oldEUID
	}
}

func SetLogFatal(f func(...any)) func() {
	old := logFatal
	logFatal = f
	return func() { logFatal = old }
}

// PanicOnLogFatal stops the caller the way os.Exit would, so specs can assert on the fatal message.
func PanicOnLogFatal() func() {
	return SetLogFatal(func(args ...any) {
		panic(fmt.Sprint(args...)) //nolint:forbidigo
	})
}

var UnknownConfigKeys = unknownConfigKeys

var SuggestOptions = suggestOptions
