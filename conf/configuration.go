package conf

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kr/pretty"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

type configOptions struct {
	ConfigFile                   string
	Address                      string
	Port                         int
	UnixSocketPerm               string
	MusicFolder                  string
	DataFolder                   string
	CacheFolder                  string
	DbPath                       string
	LogLevel                     string
	ScanInterval                 time.Duration
	ScanSchedule                 string
	SessionTimeout               time.Duration
	BaseURL                      string
	BasePath                     string
	BaseHost                     string
	BaseScheme                   string
	TLSCert                      string
	TLSKey                       string
	UILoginBackgroundURL         string
	UIWelcomeMessage             string
	MaxSidebarPlaylists          int
	EnableTranscodingConfig      bool
	EnableDownloads              bool
	EnableExternalServices       bool
	EnableMediaFileCoverArt      bool
	TranscodingCacheSize         string
	ImageCacheSize               string
	EnableArtworkPrecache        bool
	AutoImportPlaylists          bool
	PlaylistsPath                string
	AutoTranscodeDownload        bool
	DefaultDownsamplingFormat    string
	SearchFullString             bool
	RecentlyAddedByModTime       bool
	PreferSortTags               bool
	IgnoredArticles              string
	IndexGroups                  string
	SubsonicArtistParticipations bool
	FFmpegPath                   string
	MPVPath                      string
	CoverArtPriority             string
	CoverJpegQuality             int
	ArtistArtPriority            string
	EnableGravatar               bool
	EnableFavourites             bool
	EnableStarRating             bool
	EnableUserEditing            bool
	EnableSharing                bool
	DefaultDownloadableShare     bool
	DefaultTheme                 string
	DefaultLanguage              string
	DefaultUIVolume              int
	EnableReplayGain             bool
	EnableCoverAnimation         bool
	GATrackingID                 string
	EnableLogRedacting           bool
	AuthRequestLimit             int
	AuthWindowLength             time.Duration
	PasswordEncryptionKey        string
	ReverseProxyUserHeader       string
	ReverseProxyWhitelist        string
	Prometheus                   prometheusOptions
	Scanner                      scannerOptions
	Jukebox                      jukeboxOptions

	Agents       string
	LastFM       lastfmOptions
	Spotify      spotifyOptions
	ListenBrainz listenBrainzOptions

	// DevFlags. These are used to enable/disable debugging and incomplete features
	DevLogSourceLine                 bool
	DevLogLevels                     map[string]string
	DevEnableProfiler                bool
	DevAutoCreateAdminPassword       string
	DevAutoLoginUsername             string
	DevActivityPanel                 bool
	DevSidebarPlaylists              bool
	DevEnableBufferedScrobble        bool
	DevShowArtistPage                bool
	DevOffsetOptimize                int
	DevArtworkMaxRequests            int
	DevArtworkThrottleBacklogLimit   int
	DevArtworkThrottleBacklogTimeout time.Duration
	DevArtistInfoTimeToLive          time.Duration
	DevAlbumInfoTimeToLive           time.Duration
	DevLyricsTimeToLive              time.Duration
}

type scannerOptions struct {
	Extractor          string
	GenreSeparators    string
	GroupAlbumReleases bool
}

type lastfmOptions struct {
	Enabled  bool
	ApiKey   string
	Secret   string
	Language string
}

type spotifyOptions struct {
	ID     string
	Secret string
}

type listenBrainzOptions struct {
	Enabled bool
	BaseURL string
}

type prometheusOptions struct {
	Enabled     bool
	MetricsPath string
}

type AudioDeviceDefinition []string

type jukeboxOptions struct {
	Enabled bool
	Devices []AudioDeviceDefinition
	Default string
}

var (
	Server = &configOptions{}
	hooks  []func()
)

func LoadFromFile(confFile string) {
	viper.SetConfigFile(confFile)
	err := viper.ReadInConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error reading config file:", err)
		os.Exit(1)
	}
	Load()
}

func Load() {
	err := viper.Unmarshal(&Server)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error parsing config:", err)
		os.Exit(1)
	}
	err = os.MkdirAll(Server.DataFolder, os.ModePerm)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error creating data path:", "path", Server.DataFolder, err)
		os.Exit(1)
	}

	if Server.CacheFolder == "" {
		Server.CacheFolder = filepath.Join(Server.DataFolder, "cache")
	}
	err = os.MkdirAll(Server.CacheFolder, os.ModePerm)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error creating cache path:", "path", Server.CacheFolder, err)
		os.Exit(1)
	}

	Server.ConfigFile = viper.GetViper().ConfigFileUsed()
	if Server.DbPath == "" {
		Server.DbPath = filepath.Join(Server.DataFolder, consts.DefaultDbPath)
	}

	log.SetLevelString(Server.LogLevel)
	log.SetLogLevels(Server.DevLogLevels)
	log.SetLogSourceLine(Server.DevLogSourceLine)
	log.SetRedacting(Server.EnableLogRedacting)

	if err := validateScanSchedule(); err != nil {
		os.Exit(1)
	}

	if Server.BaseURL != "" {
		u, err := url.Parse(Server.BaseURL)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "FATAL: Invalid BaseURL %s: %s\n", Server.BaseURL, err.Error())
			os.Exit(1)
		}
		Server.BasePath = u.Path
		u.Path = ""
		u.RawQuery = ""
		Server.BaseHost = u.Host
		Server.BaseScheme = u.Scheme
	}

	// Print current configuration if log level is Debug
	if log.IsGreaterOrEqualTo(log.LevelDebug) {
		prettyConf := pretty.Sprintf("Loaded configuration from '%s': %# v", Server.ConfigFile, Server)
		if Server.EnableLogRedacting {
			prettyConf = log.Redact(prettyConf)
		}
		_, _ = fmt.Fprintln(os.Stderr, prettyConf)
	}

	if !Server.EnableExternalServices {
		disableExternalServices()
	}

	// Call init hooks
	for _, hook := range hooks {
		hook()
	}
}

func disableExternalServices() {
	log.Info("All external integrations are DISABLED!")
	Server.LastFM.Enabled = false
	Server.Spotify.ID = ""
	Server.ListenBrainz.Enabled = false
	Server.Agents = "filesystem"
	if Server.UILoginBackgroundURL == consts.DefaultUILoginBackgroundURL {
		Server.UILoginBackgroundURL = consts.DefaultUILoginBackgroundURLOffline
	}
}

func validateScanSchedule() error {
	if Server.ScanInterval != -1 {
		log.Warn("ScanInterval is DEPRECATED. Please use ScanSchedule. See docs at https://navidrome.org/docs/usage/configuration-options/")
		if Server.ScanSchedule != "@every 1m" {
			log.Error("You cannot specify both ScanInterval and ScanSchedule, ignoring ScanInterval")
		} else {
			if Server.ScanInterval == 0 {
				Server.ScanSchedule = ""
			} else {
				Server.ScanSchedule = fmt.Sprintf("@every %s", Server.ScanInterval)
			}
			log.Warn("Setting ScanSchedule", "schedule", Server.ScanSchedule)
		}
	}
	if Server.ScanSchedule == "0" || Server.ScanSchedule == "" {
		Server.ScanSchedule = ""
		return nil
	}
	if _, err := time.ParseDuration(Server.ScanSchedule); err == nil {
		Server.ScanSchedule = "@every " + Server.ScanSchedule
	}
	c := cron.New()
	_, err := c.AddFunc(Server.ScanSchedule, func() {})
	if err != nil {
		log.Error("Invalid ScanSchedule. Please read format spec at https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format", "schedule", Server.ScanSchedule, err)
	}
	return err
}

// AddHook is used to register initialization code that should run as soon as the config is loaded
func AddHook(hook func()) {
	hooks = append(hooks, hook)
}

func init() {
	viper.SetDefault("musicfolder", filepath.Join(".", "music"))
	viper.SetDefault("cachefolder", "")
	viper.SetDefault("datafolder", ".")
	viper.SetDefault("loglevel", "info")
	viper.SetDefault("address", "0.0.0.0")
	viper.SetDefault("port", 4533)
	viper.SetDefault("unixsocketperm", "0660")
	viper.SetDefault("sessiontimeout", consts.DefaultSessionTimeout)
	viper.SetDefault("scaninterval", -1)
	viper.SetDefault("scanschedule", "@every 1m")
	viper.SetDefault("baseurl", "")
	viper.SetDefault("tlscert", "")
	viper.SetDefault("tlskey", "")
	viper.SetDefault("uiloginbackgroundurl", consts.DefaultUILoginBackgroundURL)
	viper.SetDefault("uiwelcomemessage", "")
	viper.SetDefault("maxsidebarplaylists", consts.DefaultMaxSidebarPlaylists)
	viper.SetDefault("enabletranscodingconfig", false)
	viper.SetDefault("transcodingcachesize", "100MB")
	viper.SetDefault("imagecachesize", "100MB")
	viper.SetDefault("enableartworkprecache", true)
	viper.SetDefault("autoimportplaylists", true)
	viper.SetDefault("playlistspath", consts.DefaultPlaylistsPath)
	viper.SetDefault("enabledownloads", true)
	viper.SetDefault("enableexternalservices", true)
	viper.SetDefault("enablemediafilecoverart", true)
	viper.SetDefault("autotranscodedownload", false)
	viper.SetDefault("defaultdownsamplingformat", consts.DefaultDownsamplingFormat)
	viper.SetDefault("searchfullstring", false)
	viper.SetDefault("recentlyaddedbymodtime", false)
	viper.SetDefault("prefersorttags", false)
	viper.SetDefault("ignoredarticles", "The El La Los Las Le Les Os As O A")
	viper.SetDefault("indexgroups", "A B C D E F G H I J K L M N O P Q R S T U V W X-Z(XYZ) [Unknown]([)")
	viper.SetDefault("subsonicartistparticipations", false)
	viper.SetDefault("ffmpegpath", "")
	viper.SetDefault("coverartpriority", "cover.*, folder.*, front.*, embedded, external")
	viper.SetDefault("coverjpegquality", 75)
	viper.SetDefault("artistartpriority", "artist.*, album/artist.*, external")
	viper.SetDefault("enablegravatar", false)
	viper.SetDefault("enablefavourites", true)
	viper.SetDefault("enablestarrating", true)
	viper.SetDefault("enableuserediting", true)
	viper.SetDefault("defaulttheme", "Dark")
	viper.SetDefault("defaultlanguage", "")
	viper.SetDefault("defaultuivolume", consts.DefaultUIVolume)
	viper.SetDefault("enablereplaygain", true)
	viper.SetDefault("enablecoveranimation", true)
	viper.SetDefault("gatrackingid", "")
	viper.SetDefault("enablelogredacting", true)
	viper.SetDefault("authrequestlimit", 5)
	viper.SetDefault("authwindowlength", 20*time.Second)
	viper.SetDefault("passwordencryptionkey", "")

	viper.SetDefault("reverseproxyuserheader", "Remote-User")
	viper.SetDefault("reverseproxywhitelist", "")

	viper.SetDefault("prometheus.enabled", false)
	viper.SetDefault("prometheus.metricspath", "/metrics")

	viper.SetDefault("jukebox.enabled", false)
	viper.SetDefault("jukebox.devices", []AudioDeviceDefinition{})
	viper.SetDefault("jukebox.default", "")

	viper.SetDefault("scanner.extractor", consts.DefaultScannerExtractor)
	viper.SetDefault("scanner.genreseparators", ";/,")
	viper.SetDefault("scanner.groupalbumreleases", false)

	viper.SetDefault("agents", "filesystem,lastfm,spotify,lrclib")
	viper.SetDefault("lastfm.enabled", true)
	viper.SetDefault("lastfm.language", "en")
	viper.SetDefault("lastfm.apikey", "")
	viper.SetDefault("lastfm.secret", "")
	viper.SetDefault("spotify.id", "")
	viper.SetDefault("spotify.secret", "")
	viper.SetDefault("listenbrainz.enabled", true)
	viper.SetDefault("listenbrainz.baseurl", "https://api.listenbrainz.org/1/")

	// DevFlags. These are used to enable/disable debugging and incomplete features
	viper.SetDefault("devlogsourceline", false)
	viper.SetDefault("devenableprofiler", false)
	viper.SetDefault("devautocreateadminpassword", "")
	viper.SetDefault("devautologinusername", "")
	viper.SetDefault("devactivitypanel", true)
	viper.SetDefault("enablesharing", false)
	viper.SetDefault("defaultdownloadableshare", false)
	viper.SetDefault("devenablebufferedscrobble", true)
	viper.SetDefault("devsidebarplaylists", true)
	viper.SetDefault("devshowartistpage", true)
	viper.SetDefault("devoffsetoptimize", 50000)
	viper.SetDefault("devartworkmaxrequests", max(2, runtime.NumCPU()/3))
	viper.SetDefault("devartworkthrottlebackloglimit", consts.RequestThrottleBacklogLimit)
	viper.SetDefault("devartworkthrottlebacklogtimeout", consts.RequestThrottleBacklogTimeout)
	viper.SetDefault("devartistinfotimetolive", consts.ArtistInfoTimeToLive)
	viper.SetDefault("devalbuminfotimetolive", consts.AlbumInfoTimeToLive)
	viper.SetDefault("devlyricstimetolive", consts.LyricsInfoTimeToLive)
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
	if viper.ConfigFileUsed() != "" && err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: Navidrome could not open config file: ", err)
		os.Exit(1)
	}
}

func getConfigFile(cfgFile string) string {
	if cfgFile != "" {
		return cfgFile
	}
	return os.Getenv("ND_CONFIGFILE")
}
