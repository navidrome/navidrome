package conf

import (
	"flag"
	"os"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/koding/multiconfig"
)

type nd struct {
	Port        string `default:"4533"`
	MusicFolder string `default:"./music"`
	DbPath      string `default:"./data/navidrome.db"`
	LogLevel    string `default:"info"`

	IgnoredArticles string `default:"The El La Los Las Le Les Os As O A"`
	IndexGroups     string `default:"A B C D E F G H I J K L M N O P Q R S T U V W X-Z(XYZ) [Unknown]([)"`

	DisableDownsampling bool   `default:"false"`
	DownsampleCommand   string `default:"ffmpeg -i %s -map 0:0 -b:a %bk -v 0 -f mp3 -"`
	ProbeCommand        string `default:"ffmpeg %s -f ffmetadata"`
	ScanInterval        string `default:"1m"`

	// DevFlags. These are used to enable/disable debugging and incomplete features
	DevDisableAuthentication bool `default:"false"`
	DevDisableBanner         bool `default:"false"`
}

var Server = &nd{}

func LoadFromFile(tomlFile string) {
	m := multiconfig.NewWithPath(tomlFile)
	err := m.Load(Server)
	if err == flag.ErrHelp {
		os.Exit(1)
	}
	log.SetLogLevelString(Server.LogLevel)
}

func Load() {
	if _, err := os.Stat(consts.LocalConfigFile); err == nil {
		LoadFromFile(consts.LocalConfigFile)
	}
}
