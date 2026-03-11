package conf

func ResetConf() {
	Server = &configOptions{}
}

var SetViperDefaults = setViperDefaults

var ParseLanguages = parseLanguages

var ValidateURL = validateURL

var NormalizeSearchBackend = normalizeSearchBackend
