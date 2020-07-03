package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/spf13/viper"
)

type configOptions struct {
	ConfigFile              string
	Port                    int
	MusicFolder             string
	DataFolder              string
	DbPath                  string
	LogLevel                string
	ScanInterval            time.Duration
	SessionTimeout          time.Duration
	BaseURL                 string
	UILoginBackgroundURL    string
	IgnoredArticles         string
	IndexGroups             string
	EnableTranscodingConfig bool
	TranscodingCacheSize    string
	ImageCacheSize          string
	ProbeCommand            string
	CoverArtPriority        string
	CoverJpegQuality        int

	// DevFlags. These are used to enable/disable debugging and incomplete features
	DevLogSourceLine           bool
	DevAutoCreateAdminPassword string
}

var Server = &configOptions{}

func LoadFromFile(confFile string) {
	viper.SetConfigFile(confFile)
	Load()
}

func Load() {
	err := viper.Unmarshal(&Server)
	if err != nil {
		fmt.Println("Error parsing config:", err)
		os.Exit(1)
	}
	Server.ConfigFile = viper.GetViper().ConfigFileUsed()
	if Server.DbPath == "" {
		Server.DbPath = filepath.Join(Server.DataFolder, consts.DefaultDbPath)
	}

	log.SetLevelString(Server.LogLevel)
	log.SetLogSourceLine(Server.DevLogSourceLine)
	log.Debug("Loaded configuration", "file", Server.ConfigFile, "config", fmt.Sprintf("%#v", Server))
}

func init() {
	viper.SetDefault("musicfolder", "./music")
	viper.SetDefault("datafolder", "./")
	viper.SetDefault("loglevel", "info")
	viper.SetDefault("port", 4533)
	viper.SetDefault("sessiontimeout", consts.DefaultSessionTimeout)
	viper.SetDefault("scaninterval", time.Minute)
	viper.SetDefault("baseurl", "")
	viper.SetDefault("uiloginbackgroundurl", "")
	viper.SetDefault("enabletranscodingconfig", false)
	viper.SetDefault("transcodingcachesize", "100MB")
	viper.SetDefault("imagecachesize", "100MB")

	// Config options only valid for file configuration
	viper.SetDefault("ignoredarticles", "The El La Los Las Le Les Os As O A")
	viper.SetDefault("indexgroups", "A B C D E F G H I J K L M N O P Q R S T U V W X-Z(XYZ) [Unknown]([)")
	viper.SetDefault("probecommand", "ffmpeg %s -f ffmetadata")
	viper.SetDefault("coverartpriority", "embedded, cover.*, folder.*, front.*")
	viper.SetDefault("coverjpegquality", 75)

	// DevFlags. These are used to enable/disable debugging and incomplete features
	viper.SetDefault("devlogsourceline", false)
	viper.SetDefault("devautocreateadminpassword", "")
}

func InitConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in local directory with name "navidrome" (without extension).
		viper.AddConfigPath(".")
		viper.SetConfigName("navidrome")
	}

	_ = viper.BindEnv("port")
	viper.SetEnvPrefix("ND")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
}
