package conf

import (
	"os"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/koding/multiconfig"
)

type sonic struct {
	Port        string `default:"4533"`
	MusicFolder string `default:"./music"`
	DbPath      string `default:"./data/navidrome.db"`
	LogLevel    string `default:"info"`

	IgnoredArticles string `default:"The El La Los Las Le Les Os As O A"`
	IndexGroups     string `default:"A B C D E F G H I J K L M N O P Q R S T U V W X-Z(XYZ) [Unknown]([)"`

	DisableDownsampling bool   `default:"false"`
	DownsampleCommand   string `default:"ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -f mp3 -"`
	ProbeCommand        string `default:"ffmpeg %s -f ffmetadata"`
	PlsIgnoreFolders    bool   `default:"true"`
	PlsIgnoredPatterns  string `default:"^iCloud;\\~"`
	ScanInterval        string `default:"5m"`

	// DevFlags. These are used to enable/disable debugging and incomplete features
	DevDisableAuthentication bool   `default:"false"`
	DevDisableBanner         bool   `default:"false"`
	DevInitialPassword       string `default:""`
}

var Sonic *sonic

func LoadFromFlags() {
	l := &multiconfig.FlagLoader{}
	l.Load(Sonic)
}

func LoadFromEnv() {
	port := os.Getenv("PORT")
	if port != "" {
		Sonic.Port = port
	}
	l := &multiconfig.EnvironmentLoader{}
	err := l.Load(Sonic)
	if err != nil {
		log.Error("Error parsing configuration from environment")
	}
}

func LoadFromTags() {
	l := &multiconfig.TagLoader{}
	l.Load(Sonic)
}

func LoadFromFile(tomlFile string) {
	l := &multiconfig.TOMLLoader{Path: tomlFile}
	err := l.Load(Sonic)
	if err != nil {
		log.Error("Error loading configuration file", "file", tomlFile, err)
	}
}

func LoadFromLocalFile() {
	if _, err := os.Stat(consts.LocalConfigFile); err == nil {
		LoadFromFile(consts.LocalConfigFile)
	}
}

func Load() {
	LoadFromLocalFile()
	LoadFromEnv()
	LoadFromFlags()
	log.SetLogLevelString(Sonic.LogLevel)
}

func init() {
	Sonic = new(sonic)
	LoadFromTags()
}
