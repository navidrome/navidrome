package conf

func ResetConf() {
	Server = &configOptions{}
}

var SetViperDefaults = setViperDefaults

var ParseLanguages = parseLanguages

var ValidateURL = validateURL

var NormalizeSearchBackend = normalizeSearchBackend

var ToPascalCase = toPascalCase

var ValidateMaxImageUploadSize = validateMaxImageUploadSize

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
