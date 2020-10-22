package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/spf13/viper"
)

type configOptions struct {
	ConfigFile              string
	Address                 string
	Port                    int
	MusicFolder             string
	DataFolder              string
	DbPath                  string
	LogLevel                string
	ScanInterval            time.Duration
	SessionTimeout          time.Duration
	BaseURL                 string
	UILoginBackgroundURL    string
	EnableTranscodingConfig bool
	TranscodingCacheSize    string
	ImageCacheSize          string
	AutoImportPlaylists     bool

	SearchFullString bool
	IgnoredArticles  string
	IndexGroups      string
	ProbeCommand     string
	CoverArtPriority string
	CoverJpegQuality int
	UIWelcomeMessage string
	GATrackingID     string
	AuthRequestLimit int
	AuthWindowLength time.Duration

	Scanner scannerOptions
	LastFM  lastfmOptions
	Spotify spotifyOptions
	LDAP    ldapOptions

	// DevFlags. These are used to enable/disable debugging and incomplete features
	DevLogSourceLine           bool
	DevAutoCreateAdminPassword string
}

type scannerOptions struct {
	Extractor string
}

type lastfmOptions struct {
	ApiKey   string
	Secret   string
	Language string
}

type spotifyOptions struct {
	ID     string
	Secret string
}

type ldapOptions struct {
	Host         string
	BindDN       string
	BindPassword string
	Base         string
	SearchFilter string
	Mail         string
	Name         string
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
	err = os.MkdirAll(Server.DataFolder, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating data path:", "path", Server.DataFolder, err)
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
	viper.SetDefault("musicfolder", filepath.Join(".", "music"))
	viper.SetDefault("datafolder", ".")
	viper.SetDefault("loglevel", "info")
	viper.SetDefault("address", "0.0.0.0")
	viper.SetDefault("port", 4533)
	viper.SetDefault("sessiontimeout", consts.DefaultSessionTimeout)
	viper.SetDefault("scaninterval", time.Minute)
	viper.SetDefault("baseurl", "")
	viper.SetDefault("uiloginbackgroundurl", "https://source.unsplash.com/random/1600x900?music")
	viper.SetDefault("enabletranscodingconfig", false)
	viper.SetDefault("transcodingcachesize", "100MB")
	viper.SetDefault("imagecachesize", "100MB")
	viper.SetDefault("autoimportplaylists", true)

	// Config options only valid for file/env configuration
	viper.SetDefault("searchfullstring", false)
	viper.SetDefault("ignoredarticles", "The El La Los Las Le Les Os As O A")
	viper.SetDefault("indexgroups", "A B C D E F G H I J K L M N O P Q R S T U V W X-Z(XYZ) [Unknown]([)")
	viper.SetDefault("probecommand", "ffmpeg %s -f ffmetadata")
	viper.SetDefault("coverartpriority", "embedded, cover.*, folder.*, front.*")
	viper.SetDefault("coverjpegquality", 75)
	viper.SetDefault("uiwelcomemessage", "")
	viper.SetDefault("gatrackingid", "")
	viper.SetDefault("authrequestlimit", 5)
	viper.SetDefault("authwindowlength", 20*time.Second)

	viper.SetDefault("scanner.extractor", "taglib")
	viper.SetDefault("lastfm.language", "en")
	viper.SetDefault("lastfm.apikey", "")
	viper.SetDefault("lastfm.secret", "")
	viper.SetDefault("spotify.id", "")
	viper.SetDefault("spotify.secret", "")

	viper.SetDefault("ldap.host", "ldap://localhost:389")
	viper.SetDefault("ldap.binddn", "")
	viper.SetDefault("ldap.bindpassword", "")
	viper.SetDefault("ldap.base", "")
	viper.SetDefault("ldap.searchfilter", "(&(objectClass=inetOrgPerson)(uid=%s))")
	viper.SetDefault("ldap.mail", "mail")
	viper.SetDefault("ldap.name", "cn")

	// DevFlags. These are used to enable/disable debugging and incomplete features
	viper.SetDefault("devlogsourceline", false)
	viper.SetDefault("devautocreateadminpassword", "")
	viper.SetDefault("devoldscanner", false)
}

func InitConfig(cfgFile string) {
	cfgFile = getConfigFile(cfgFile)
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
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if cfgFile != "" && err != nil {
		fmt.Println("Navidrome could not open config file: ", err)
		os.Exit(1)
	}
}

func getConfigFile(cfgFile string) string {
	if cfgFile != "" {
		return cfgFile
	}
	return os.Getenv("ND_CONFIGFILE")
}
