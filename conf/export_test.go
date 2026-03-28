package conf

func ResetConf() {
	Server = &configOptions{}
}

var SetViperDefaults = setViperDefaults

var ParseLanguages = parseLanguages

var ValidateURL = validateURL

var NormalizeSearchBackend = normalizeSearchBackend

var ToPascalCase = toPascalCase

func SetFatalFunc(f func(string)) func() {
	old := fatalFunc
	fatalFunc = f
	return func() { fatalFunc = old }
}
