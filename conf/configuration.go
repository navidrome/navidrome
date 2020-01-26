package conf

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/koding/multiconfig"
)

type nd struct {
	Port        string `default:"4533"`
	MusicFolder string `default:"./music"`
	DataFolder  string `default:"./"`
	DbPath      string
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

func LoadFromFile(confFile string) {
	m := multiconfig.NewWithPath(confFile)
	err := m.Load(Server)
	if err == flag.ErrHelp {
		os.Exit(1)
	}
	if Server.DbPath == "" {
		Server.DbPath = filepath.Join(Server.DataFolder, "navidrome.db")
	}
	log.SerLevelString(Server.LogLevel)
	log.Trace("Loaded configuration", "file", confFile, "config", fmt.Sprintf("%#v", Server))
}

func Load() {
	if _, err := os.Stat(consts.LocalConfigFile); err == nil {
		LoadFromFile(consts.LocalConfigFile)
	}
}
