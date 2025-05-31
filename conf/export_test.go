package conf

func ResetConf() {
	Server = &configOptions{}
}

var SetViperDefaults = setViperDefaults
