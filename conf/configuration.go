package conf

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/kr/pretty"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/chain"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

type configOptions struct {
	ConfigFile                      string
	Address                         string
	Port                            int
	UnixSocketPerm                  string
	MusicFolder                     string
	DataFolder                      string
	CacheFolder                     string
	DbPath                          string
	LogLevel                        string
	LogFile                         string
	SessionTimeout                  time.Duration
	BaseURL                         string
	BasePath                        string
	BaseHost                        string
	BaseScheme                      string
	TLSCert                         string
	TLSKey                          string
	UILoginBackgroundURL            string
	UIWelcomeMessage                string
	MaxSidebarPlaylists             int
	EnableTranscodingConfig         bool
	EnableDownloads                 bool
	EnableExternalServices          bool
	EnableInsightsCollector         bool
	EnableMediaFileCoverArt         bool
	TranscodingCacheSize            string
	ImageCacheSize                  string
	AlbumPlayCountMode              string
	EnableArtworkPrecache           bool
	AutoImportPlaylists             bool
	DefaultPlaylistPublicVisibility bool
	PlaylistsPath                   string
	SmartPlaylistRefreshDelay       time.Duration
	AutoTranscodeDownload           bool
	DefaultDownsamplingFormat       string
	SearchFullString                bool
	RecentlyAddedByModTime          bool
	PreferSortTags                  bool
	IgnoredArticles                 string
	IndexGroups                     string
	SubsonicArtistParticipations    bool
	DefaultReportRealPath           bool
	FFmpegPath                      string
	MPVPath                         string
	MPVCmdTemplate                  string
	CoverArtPriority                string
	CoverJpegQuality                int
	ArtistArtPriority               string
	EnableGravatar                  bool
	EnableFavourites                bool
	EnableStarRating                bool
	EnableUserEditing               bool
	EnableSharing                   bool
	ShareURL                        string
	DefaultDownloadableShare        bool
	DefaultTheme                    string
	DefaultLanguage                 string
	DefaultUIVolume                 int
	EnableReplayGain                bool
	EnableCoverAnimation            bool
	GATrackingID                    string
	EnableLogRedacting              bool
	AuthRequestLimit                int
	AuthWindowLength                time.Duration
	PasswordEncryptionKey           string
	ReverseProxyUserHeader          string
	ReverseProxyWhitelist           string
	HTTPSecurityHeaders             secureOptions
	Prometheus                      prometheusOptions
	Scanner                         scannerOptions
	Jukebox                         jukeboxOptions
	Backup                          backupOptions
	PID                             pidOptions
	Inspect                         inspectOptions

	Agents       string
	LastFM       lastfmOptions
	Spotify      spotifyOptions
	ListenBrainz listenBrainzOptions
	Tags         map[string]TagConf

	// DevFlags. These are used to enable/disable debugging and incomplete features
	DevLogSourceLine                 bool
	DevLogLevels                     map[string]string
	DevEnableProfiler                bool
	DevAutoCreateAdminPassword       string
	DevAutoLoginUsername             string
	DevActivityPanel                 bool
	DevActivityPanelUpdateRate       time.Duration
	DevSidebarPlaylists              bool
	DevEnableBufferedScrobble        bool
	DevShowArtistPage                bool
	DevOffsetOptimize                int
	DevArtworkMaxRequests            int
	DevArtworkThrottleBacklogLimit   int
	DevArtworkThrottleBacklogTimeout time.Duration
	DevArtistInfoTimeToLive          time.Duration
	DevAlbumInfoTimeToLive           time.Duration
	DevExternalScanner               bool
	DevScannerThreads                uint
	DevInsightsInitialDelay          time.Duration
	DevEnablePlayerInsights          bool
	DevOpenSubsonicDisabledClients   string
}

type scannerOptions struct {
	Enabled            bool
	Schedule           string
	WatcherWait        time.Duration
	ScanOnStartup      bool
	Extractor          string
	GroupAlbumReleases bool // Deprecated: BFR Update docs
}

type TagConf struct {
	Aliases   []string `yaml:"aliases"`
	Type      string   `yaml:"type"`
	MaxLength int      `yaml:"maxLength"`
	Split     []string `yaml:"split"`
	Album     bool     `yaml:"album"`
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

type secureOptions struct {
	CustomFrameOptionsValue string
}

type prometheusOptions struct {
	Enabled     bool
	MetricsPath string
	Password    string
}

type AudioDeviceDefinition []string

type jukeboxOptions struct {
	Enabled   bool
	Devices   []AudioDeviceDefinition
	Default   string
	AdminOnly bool
}

type backupOptions struct {
	Count    int
	Path     string
	Schedule string
}

type pidOptions struct {
	Track string
	Album string
}

type inspectOptions struct {
	Enabled        bool
	MaxRequests    int
	BacklogLimit   int
	BacklogTimeout int
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
	Load(true)
}

func Load(noConfigDump bool) {
	parseIniFileConfiguration()

	err := viper.Unmarshal(&Server)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error parsing config:", err)
		os.Exit(1)
	}

	err = os.MkdirAll(Server.DataFolder, os.ModePerm)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error creating data path:", err)
		os.Exit(1)
	}

	if Server.CacheFolder == "" {
		Server.CacheFolder = filepath.Join(Server.DataFolder, "cache")
	}
	err = os.MkdirAll(Server.CacheFolder, os.ModePerm)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error creating cache path:", err)
		os.Exit(1)
	}

	Server.ConfigFile = viper.GetViper().ConfigFileUsed()
	if Server.DbPath == "" {
		Server.DbPath = filepath.Join(Server.DataFolder, consts.DefaultDbPath)
	}

	if Server.Backup.Path != "" {
		err = os.MkdirAll(Server.Backup.Path, os.ModePerm)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error creating backup path:", err)
			os.Exit(1)
		}
	}

	out := os.Stderr
	if Server.LogFile != "" {
		out, err = os.OpenFile(Server.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "FATAL: Error opening log file %s: %s\n", Server.LogFile, err.Error())
			os.Exit(1)
		}
		log.SetOutput(out)
	}

	log.SetLevelString(Server.LogLevel)
	log.SetLogLevels(Server.DevLogLevels)
	log.SetLogSourceLine(Server.DevLogSourceLine)
	log.SetRedacting(Server.EnableLogRedacting)

	err = chain.RunSequentially(
		validateScanSchedule,
		validateBackupSchedule,
		validatePlaylistsPath,
	)
	if err != nil {
		os.Exit(1)
	}

	if Server.BaseURL != "" {
		u, err := url.Parse(Server.BaseURL)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "FATAL: Invalid BaseURL:", err)
			os.Exit(1)
		}
		Server.BasePath = u.Path
		u.Path = ""
		u.RawQuery = ""
		Server.BaseHost = u.Host
		Server.BaseScheme = u.Scheme
	}

	// Print current configuration if log level is Debug
	if log.IsGreaterOrEqualTo(log.LevelDebug) && !noConfigDump {
		prettyConf := pretty.Sprintf("Loaded configuration from '%s': %# v", Server.ConfigFile, Server)
		if Server.EnableLogRedacting {
			prettyConf = log.Redact(prettyConf)
		}
		_, _ = fmt.Fprintln(out, prettyConf)
	}

	if !Server.EnableExternalServices {
		disableExternalServices()
	}

	// BFR Remove before release
	if Server.Scanner.Extractor != consts.DefaultScannerExtractor {
		log.Warn(fmt.Sprintf("Extractor '%s' is not implemented, using 'taglib'", Server.Scanner.Extractor))
		Server.Scanner.Extractor = consts.DefaultScannerExtractor
	}

	// Call init hooks
	for _, hook := range hooks {
		hook()
	}
}

// parseIniFileConfiguration is used to parse the config file when it is in INI format. For INI files, it
// would require a nested structure, so instead we unmarshal it to a map and then merge the nested [default]
// section into the root level.
func parseIniFileConfiguration() {
	cfgFile := viper.ConfigFileUsed()
	if strings.ToLower(filepath.Ext(cfgFile)) == ".ini" {
		var iniConfig map[string]interface{}
		err := viper.Unmarshal(&iniConfig)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error parsing config:", err)
			os.Exit(1)
		}
		cfg, ok := iniConfig["default"].(map[string]any)
		if !ok {
			_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error parsing config: missing [default] section:", iniConfig)
			os.Exit(1)
		}
		err = viper.MergeConfigMap(cfg)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "FATAL: Error parsing config:", err)
			os.Exit(1)
		}
	}
}

func disableExternalServices() {
	log.Info("All external integrations are DISABLED!")
	Server.EnableInsightsCollector = false
	Server.LastFM.Enabled = false
	Server.Spotify.ID = ""
	Server.ListenBrainz.Enabled = false
	Server.Agents = ""
	if Server.UILoginBackgroundURL == consts.DefaultUILoginBackgroundURL {
		Server.UILoginBackgroundURL = consts.DefaultUILoginBackgroundURLOffline
	}
}

func validatePlaylistsPath() error {
	for _, path := range strings.Split(Server.PlaylistsPath, string(filepath.ListSeparator)) {
		_, err := doublestar.Match(path, "")
		if err != nil {
			log.Error("Invalid PlaylistsPath", "path", path, err)
			return err
		}
	}
	return nil
}

func validateScanSchedule() error {
	if Server.Scanner.Schedule == "0" || Server.Scanner.Schedule == "" {
		Server.Scanner.Schedule = ""
		return nil
	}
	var err error
	Server.Scanner.Schedule, err = validateSchedule(Server.Scanner.Schedule, "Scanner.Schedule")
	return err
}

func validateBackupSchedule() error {
	if Server.Backup.Path == "" || Server.Backup.Schedule == "" || Server.Backup.Count == 0 {
		Server.Backup.Schedule = ""
		return nil
	}
	var err error
	Server.Backup.Schedule, err = validateSchedule(Server.Backup.Schedule, "Backup.Schedule")
	return err
}

func validateSchedule(schedule, field string) (string, error) {
	if _, err := time.ParseDuration(schedule); err == nil {
		schedule = "@every " + schedule
	}
	c := cron.New()
	id, err := c.AddFunc(schedule, func() {})
	if err != nil {
		log.Error(fmt.Sprintf("Invalid %s. Please read format spec at https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format", field), "schedule", schedule, err)
	} else {
		c.Remove(id)
	}
	return schedule, err
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
	viper.SetDefault("logfile", "")
	viper.SetDefault("address", "0.0.0.0")
	viper.SetDefault("port", 4533)
	viper.SetDefault("unixsocketperm", "0660")
	viper.SetDefault("sessiontimeout", consts.DefaultSessionTimeout)
	viper.SetDefault("baseurl", "")
	viper.SetDefault("tlscert", "")
	viper.SetDefault("tlskey", "")
	viper.SetDefault("uiloginbackgroundurl", consts.DefaultUILoginBackgroundURL)
	viper.SetDefault("uiwelcomemessage", "")
	viper.SetDefault("maxsidebarplaylists", consts.DefaultMaxSidebarPlaylists)
	viper.SetDefault("enabletranscodingconfig", false)
	viper.SetDefault("transcodingcachesize", "100MB")
	viper.SetDefault("imagecachesize", "100MB")
	viper.SetDefault("albumplaycountmode", consts.AlbumPlayCountModeAbsolute)
	viper.SetDefault("enableartworkprecache", true)
	viper.SetDefault("autoimportplaylists", true)
	viper.SetDefault("defaultplaylistpublicvisibility", false)
	viper.SetDefault("playlistspath", "")
	viper.SetDefault("smartPlaylistRefreshDelay", 5*time.Second)
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
	viper.SetDefault("defaultreportrealpath", false)
	viper.SetDefault("ffmpegpath", "")
	viper.SetDefault("mpvcmdtemplate", "mpv --audio-device=%d --no-audio-display --pause %f --input-ipc-server=%s")

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
	viper.SetDefault("enablesharing", false)
	viper.SetDefault("shareurl", "")
	viper.SetDefault("defaultdownloadableshare", false)
	viper.SetDefault("gatrackingid", "")
	viper.SetDefault("enableinsightscollector", true)
	viper.SetDefault("enablelogredacting", true)
	viper.SetDefault("authrequestlimit", 5)
	viper.SetDefault("authwindowlength", 20*time.Second)
	viper.SetDefault("passwordencryptionkey", "")

	viper.SetDefault("reverseproxyuserheader", "Remote-User")
	viper.SetDefault("reverseproxywhitelist", "")

	viper.SetDefault("prometheus.enabled", false)
	viper.SetDefault("prometheus.metricspath", consts.PrometheusDefaultPath)
	viper.SetDefault("prometheus.password", "")

	viper.SetDefault("jukebox.enabled", false)
	viper.SetDefault("jukebox.devices", []AudioDeviceDefinition{})
	viper.SetDefault("jukebox.default", "")
	viper.SetDefault("jukebox.adminonly", true)

	viper.SetDefault("scanner.enabled", true)
	viper.SetDefault("scanner.schedule", "0")
	viper.SetDefault("scanner.extractor", consts.DefaultScannerExtractor)
	viper.SetDefault("scanner.groupalbumreleases", false)
	viper.SetDefault("scanner.watcherwait", consts.DefaultWatcherWait)
	viper.SetDefault("scanner.scanonstartup", true)

	viper.SetDefault("agents", "lastfm,spotify")
	viper.SetDefault("lastfm.enabled", true)
	viper.SetDefault("lastfm.language", "en")
	viper.SetDefault("lastfm.apikey", "")
	viper.SetDefault("lastfm.secret", "")
	viper.SetDefault("spotify.id", "")
	viper.SetDefault("spotify.secret", "")
	viper.SetDefault("listenbrainz.enabled", true)
	viper.SetDefault("listenbrainz.baseurl", "https://api.listenbrainz.org/1/")

	viper.SetDefault("httpsecurityheaders.customframeoptionsvalue", "DENY")

	viper.SetDefault("backup.path", "")
	viper.SetDefault("backup.schedule", "")
	viper.SetDefault("backup.count", 0)

	viper.SetDefault("pid.track", consts.DefaultTrackPID)
	viper.SetDefault("pid.album", consts.DefaultAlbumPID)

	viper.SetDefault("inspect.enabled", true)
	viper.SetDefault("inspect.maxrequests", 1)
	viper.SetDefault("inspect.backloglimit", consts.RequestThrottleBacklogLimit)
	viper.SetDefault("inspect.backlogtimeout", consts.RequestThrottleBacklogTimeout)

	// DevFlags. These are used to enable/disable debugging and incomplete features
	viper.SetDefault("devlogsourceline", false)
	viper.SetDefault("devenableprofiler", false)
	viper.SetDefault("devautocreateadminpassword", "")
	viper.SetDefault("devautologinusername", "")
	viper.SetDefault("devactivitypanel", true)
	viper.SetDefault("devactivitypanelupdaterate", 300*time.Millisecond)
	viper.SetDefault("devenablebufferedscrobble", true)
	viper.SetDefault("devsidebarplaylists", true)
	viper.SetDefault("devshowartistpage", true)
	viper.SetDefault("devoffsetoptimize", 50000)
	viper.SetDefault("devartworkmaxrequests", max(2, runtime.NumCPU()/3))
	viper.SetDefault("devartworkthrottlebackloglimit", consts.RequestThrottleBacklogLimit)
	viper.SetDefault("devartworkthrottlebacklogtimeout", consts.RequestThrottleBacklogTimeout)
	viper.SetDefault("devartistinfotimetolive", consts.ArtistInfoTimeToLive)
	viper.SetDefault("devalbuminfotimetolive", consts.AlbumInfoTimeToLive)
	viper.SetDefault("devexternalscanner", true)
	viper.SetDefault("devscannerthreads", 5)
	viper.SetDefault("devinsightsinitialdelay", consts.InsightsInitialDelay)
	viper.SetDefault("devenableplayerinsights", true)
	viper.SetDefault("devopensubsonicdisabledclients", "DSub")
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
