package conf

import (
	"cmp"
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dustin/go-humanize"
	"github.com/go-viper/encoding/ini"
	"github.com/go-viper/mapstructure/v2"
	"github.com/kr/pretty"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/scheduler"
	"github.com/navidrome/navidrome/utils/run"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/spf13/viper"
)

type configOptions struct {
	ConfigFile                      string `conf:"-"`
	Address                         string
	Port                            int
	UnixSocketPerm                  string
	EnforceNonRootUser              bool
	MusicFolder                     string
	DataFolder                      Dir
	CacheFolder                     Dir
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
	EnableMediaFileDeletion         bool
	EnableExternalServices          bool
	EnableM3UExternalAlbumArt       bool
	EnableInsightsCollector         bool
	EnableScheduledDBAnalyze        bool
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
	Search                          searchOptions  `json:",omitzero"`
	Matcher                         matcherOptions `json:",omitzero"`
	RecentlyAddedByModTime          bool
	PreferSortTags                  bool
	IgnoredArticles                 string
	IndexGroups                     string
	FFmpegPath                      string
	MPVPath                         string
	MPVCmdTemplate                  string
	CoverArtPriority                string
	CoverArtQuality                 int
	EnableWebPEncoding              bool
	ArtistArtPriority               string
	ArtistImageFolder               string
	DiscArtPriority                 string
	LyricsPriority                  string
	EnableGravatar                  bool
	EnableFavourites                bool
	EnableStarRating                bool
	EnableUserEditing               bool
	EnableArtworkUpload             bool
	MaxImageUploadSize              string
	MaxImageSize                    string
	EnableSharing                   bool
	ShareURL                        string
	DefaultShareExpiration          time.Duration
	DefaultDownloadableShare        bool
	DefaultTheme                    string
	DefaultLanguage                 string
	DefaultUIVolume                 int
	UISearchDebounceMs              int
	UICoverArtSize                  int
	EnableReplayGain                bool
	EnableCoverAnimation            bool
	EnableNowPlaying                bool
	UIPlaybackReportInterval        time.Duration
	GATrackingID                    string
	EnableLogRedacting              bool
	AuthRequestLimit                int
	AuthWindowLength                time.Duration
	PasswordEncryptionKey           string
	ExtAuth                         extAuthOptions
	Plugins                         pluginsOptions
	HTTPHeaders                     httpHeaderOptions   `json:",omitzero"`
	Prometheus                      prometheusOptions   `json:",omitzero"`
	Scanner                         scannerOptions      `json:",omitzero"`
	Jukebox                         jukeboxOptions      `json:",omitzero"`
	Backup                          backupOptions       `json:",omitzero"`
	PID                             pidOptions          `json:",omitzero"`
	Inspect                         inspectOptions      `json:",omitzero"`
	Subsonic                        subsonicOptions     `json:",omitzero"`
	Transcoding                     transcodingOptions  `json:",omitzero"`
	LastFM                          lastfmOptions       `json:",omitzero"`
	Deezer                          deezerOptions       `json:",omitzero"`
	ListenBrainz                    listenBrainzOptions `json:",omitzero"`
	Jellyfin                        jellyfinOptions     `json:",omitzero"`
	EnableScrobbleHistory           bool
	Tags                            map[string]TagConf `json:",omitempty"`
	Agents                          string

	// DevFlags. These are used to enable/disable debugging and incomplete features
	DevLogLevels                      map[string]string `json:",omitempty"`
	DevLogSourceLine                  bool
	DevEnableProfiler                 bool
	DevAutoCreateAdminPassword        string
	DevAutoLoginUsername              string
	DevActivityPanel                  bool
	DevActivityPanelUpdateRate        time.Duration
	DevSidebarPlaylists               bool
	DevShowArtistPage                 bool
	DevUIShowConfig                   bool
	DevNewEventStream                 bool
	DevOffsetOptimize                 int
	DevArtworkMaxRequests             int
	DevArtworkThrottleBacklogLimit    int
	DevArtworkThrottleBacklogTimeout  time.Duration
	DevArtworkThrottleBuffered        bool
	DevArtworkWorkerConcurrency       int
	DevArtworkExternalMaxRPS          int
	DevArtistInfoTimeToLive           time.Duration
	DevAlbumInfoTimeToLive            time.Duration
	DevExternalScanner                bool
	DevScannerThreads                 uint
	DevSelectiveWatcher               bool
	DevInsightsInitialDelay           time.Duration
	DevEnablePlayerInsights           bool
	DevEnablePluginsInsights          bool
	DevPluginCompilationTimeout       time.Duration
	DevExternalArtistFetchMultiplier  float64
	DevPreserveUnicodeInExternalCalls bool
	DevEnableMediaFileProbe           bool
}

type scannerOptions struct {
	Enabled               bool
	Schedule              string
	WatcherWait           time.Duration
	ScanOnStartup         bool
	Extractor             string
	ArtistJoiner          string
	ArtistSplitExceptions []string // Artist names never split by tag separators
	GenreSeparators       string   // Deprecated: Use Tags.genre.Split instead
	GroupAlbumReleases    bool     // Deprecated: Use PID.Album instead
	FollowSymlinks        bool     // Whether to follow symlinks when scanning directories
	IgnoreDotFolders      bool     // Whether to ignore folders whose name starts with a dot when scanning
	PurgeMissing          string   // Values: "never", "always", "full"
}

type transcodingOptions struct {
	MaxConcurrent        int
	MaxConcurrentPerUser int
	EnableCancellation   bool
}

type subsonicOptions struct {
	AppendSubtitle        bool
	AppendAlbumVersion    bool
	ArtistParticipations  bool
	DefaultReportRealPath bool
	EnableAverageRating   bool
	LegacyClients         string
	MinimalClients        string
}

type TagConf struct {
	Ignore    bool     `yaml:"ignore" json:",omitempty"`
	Aliases   []string `yaml:"aliases" json:",omitempty"`
	Type      string   `yaml:"type" json:",omitempty"`
	MaxLength int      `yaml:"maxLength" json:",omitempty"`
	Split     []string `yaml:"split" json:",omitempty"`
	Album     bool     `yaml:"album" json:",omitempty"`
}

type lastfmOptions struct {
	Enabled                 bool
	ApiKey                  string //nolint:gosec
	Secret                  string //nolint:gosec
	Language                string
	ScrobbleFirstArtistOnly bool

	// Computed values
	Languages []string `conf:"-"` // Computed from Language, split by comma
}

type deezerOptions struct {
	Enabled  bool
	Language string

	// Computed values
	Languages []string `conf:"-"` // Computed from Language, split by comma
}

type listenBrainzOptions struct {
	Enabled         bool
	BaseURL         string
	ArtistAlgorithm string
	TrackAlgorithm  string
}

type jellyfinOptions struct {
	Enabled    bool
	ServerName string
	// ExposedPublicUsers is a comma-separated list of usernames to advertise on the unauthenticated
	// GET /Users/Public, so Jellyfin clients can show a login user-picker. Empty exposes no users.
	ExposedPublicUsers string
	// MaxConcurrentStreams bounds how many collection responses can stream at once. Each holds a DB
	// cursor — and its pooled connection — for the whole client-paced response, so without a bound
	// enough slow clients would take the entire pool and stall the scanner, scrobbles and the UI.
	MaxConcurrentStreams int
}

type httpHeaderOptions struct {
	FrameOptions string
}

type prometheusOptions struct {
	Enabled     bool
	MetricsPath string
	Password    string //nolint:gosec
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
	Path     Dir
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

type pluginsOptions struct {
	Enabled    bool
	Folder     Dir
	CacheSize  string
	AutoReload bool
	LogLevel   string
}

type extAuthOptions struct {
	TrustedSources string
	UserHeader     string
	LogoutURL      string
}

type searchOptions struct {
	Backend    string
	FullString bool
}

type matcherOptions struct {
	PreferStarred  bool
	FuzzyThreshold int
}

// logFatal prints a fatal error message to stderr and exits.
// Overridden in tests to allow testing fatal paths.
var logFatal = func(args ...any) {
	_, _ = fmt.Fprintln(os.Stderr, append([]any{"FATAL:"}, args...)...)
	os.Exit(1)
}

var getEUID = os.Geteuid

var currentGOOS = func() string {
	return runtime.GOOS
}

var (
	Server = &configOptions{}
	hooks  []func()
)

// SnapshotConfig returns a function that restores Server to its current state.
// Uses JSON round-tripping so Dir fields get fresh sync.Once values.
func SnapshotConfig() func() {
	snapshot, err := json.Marshal(Server)
	if err != nil {
		panic(fmt.Sprintf("SnapshotConfig: marshal failed: %v", err))
	}
	return func() {
		var restored configOptions
		if err := json.Unmarshal(snapshot, &restored); err != nil {
			panic(fmt.Sprintf("SnapshotConfig: unmarshal failed: %v", err))
		}
		Server = &restored
	}
}

func LoadFromFile(confFile string) {
	viper.SetConfigFile(confFile)
	err := viper.ReadInConfig()
	if err != nil {
		logFatal("Error reading config file:", err)
	}
	Load(true)
}

func Load(noConfigDump bool) {
	parseIniFileConfiguration()
	remapEnvVarKeysFromConfig()

	// Map deprecated options to their new names for backwards compatibility
	for _, o := range deprecatedOptions {
		if o.replacement != "" {
			mapDeprecatedOption(o.name, o.replacement)
		}
	}

	err := viper.Unmarshal(&Server, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.TextUnmarshallerHookFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	))
	if err != nil {
		logFatal("Error parsing config:", err)
	}

	// Validate non-root user early, before any filesystem operations
	if err := validateEnforceNonRootUser(); err != nil {
		logFatal(err)
	}

	if Server.CacheFolder.String() == "" {
		Server.CacheFolder = NewDir(filepath.Join(Server.DataFolder.String(), "cache"))
	}

	if Server.Plugins.Enabled {
		if Server.Plugins.Folder.String() == "" {
			Server.Plugins.Folder = NewDirWithPerm(filepath.Join(Server.DataFolder.String(), "plugins"), 0700)
		} else {
			Server.Plugins.Folder = NewDirWithPerm(Server.Plugins.Folder.String(), 0700)
		}
	}

	Server.ConfigFile = viper.GetViper().ConfigFileUsed()
	if Server.DbPath == "" {
		Server.DbPath = filepath.Join(Server.DataFolder.String(), consts.DefaultDbPath)
	}

	out := os.Stderr
	if Server.LogFile != "" {
		if mkErr := os.MkdirAll(filepath.Dir(Server.LogFile), os.ModePerm); mkErr != nil {
			logFatal(fmt.Sprintf("Error creating log file directory: %s", mkErr.Error()))
		}
		out, err = os.OpenFile(Server.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logFatal(fmt.Sprintf("Error opening log file %s: %s", Server.LogFile, err.Error()))
		}
		log.SetOutput(out)
	} else if os.Getenv("ND_SYSTEMD_PRIORITY_LOGGING") != "" && os.Getenv("JOURNAL_STREAM") != "" {
		// When running under systemd, prepend syslog priority prefixes so
		// journald assigns the correct severity to each log line.
		// Note that we have an additional environment variable, as JOURNAL_STREAM
		// can be present in a systemd environment even if not running as a systemd service
		log.EnableJournalFormat()
	}

	log.SetLevelString(Server.LogLevel)
	log.SetLogLevels(Server.DevLogLevels)
	log.SetLogSourceLine(Server.DevLogSourceLine)
	log.SetRedacting(Server.EnableLogRedacting)

	// Log deprecated, removed and unknown options
	for _, o := range deprecatedOptions {
		logDeprecatedOptions(o.name, o.replacement)
	}
	logRemovedOptions(removedOptions...)
	logUnknownOptions()

	err = run.Sequentially(
		validateScanSchedule,
		validateBackupSchedule,
		validatePlaylistsPath,
		validatePurgeMissingOption,
		validateByteSize("MaxImageUploadSize", Server.MaxImageUploadSize),
		validateByteSize("MaxImageSize", Server.MaxImageSize),
		validateURL("ExtAuth.LogoutURL", Server.ExtAuth.LogoutURL),
	)
	if err != nil {
		logFatal(err)
	}

	Server.Search.Backend = normalizeSearchBackend(Server.Search.Backend)

	if Server.BaseURL != "" {
		u, err := url.Parse(Server.BaseURL)
		if err != nil {
			logFatal("Invalid BaseURL:", err)
		}
		Server.BasePath = u.Path
		u.Path = ""
		u.RawQuery = ""
		Server.BaseHost = u.Host
		Server.BaseScheme = u.Scheme
	}

	// Log configuration source
	if Server.ConfigFile != "" {
		log.Info("Loaded configuration", "file", Server.ConfigFile)
	} else if hasNDEnvVars() {
		log.Info("No configuration file found. Loaded configuration only from environment variables")
	} else {
		log.Warn("No configuration file found. Using default values. To specify a config file, use the --configfile flag or set the ND_CONFIGFILE environment variable.")
	}

	// Print current configuration if log level is Debug
	if log.IsGreaterOrEqualTo(log.LevelDebug) && !noConfigDump {
		prettyConf := pretty.Sprintf("Configuration: %# v", Server)
		if Server.EnableLogRedacting {
			prettyConf = log.Redact(prettyConf)
		}
		_, _ = fmt.Fprintln(out, prettyConf)
	}

	if !Server.EnableExternalServices {
		disableExternalServices()
	}

	// Make sure we don't have empty PIDs
	Server.PID.Album = cmp.Or(Server.PID.Album, consts.DefaultAlbumPID)
	Server.PID.Track = cmp.Or(Server.PID.Track, consts.DefaultTrackPID)

	// Parse LastFM.Language into Languages slice (comma-separated, with fallback to DefaultInfoLanguage)
	Server.LastFM.Languages = parseLanguages(Server.LastFM.Language)

	// Parse Deezer.Language into Languages slice (comma-separated, with fallback to DefaultInfoLanguage)
	Server.Deezer.Languages = parseLanguages(Server.Deezer.Language)

	// Validate other options
	if Server.UICoverArtSize < 200 || Server.UICoverArtSize > 1200 {
		newValue := max(200, min(1200, Server.UICoverArtSize))
		log.Warn("UICoverArtSize must be between 200 and 1200, clamping", "value", Server.UICoverArtSize, "newValue", newValue)
		Server.UICoverArtSize = newValue
	}

	// Floor MaxImageSize at MaxImageUploadSize so accepted uploads can always be read back.
	imgSize, _ := humanize.ParseBytes(Server.MaxImageSize)
	uploadSize, _ := humanize.ParseBytes(Server.MaxImageUploadSize)
	if imgSize < uploadSize {
		log.Warn("MaxImageSize must be at least MaxImageUploadSize, raising", "value", Server.MaxImageSize, "newValue", Server.MaxImageUploadSize)
		Server.MaxImageSize = Server.MaxImageUploadSize
	}

	// Call init hooks
	for _, hook := range hooks {
		hook()
	}
}

// deprecatedOptions still work, but will be removed in a future release. An empty
// replacement means the option is now ignored.
var deprecatedOptions = []struct{ name, replacement string }{
	{"Scanner.GenreSeparators", ""},
	{"Scanner.GroupAlbumReleases", ""},
	{"DevEnableBufferedScrobble", ""},
	{"SearchFullString", "Search.FullString"},
	{"ReverseProxyWhitelist", "ExtAuth.TrustedSources"},
	{"ReverseProxyUserHeader", "ExtAuth.UserHeader"},
	{"HTTPSecurityHeaders.CustomFrameOptionsValue", "HTTPHeaders.FrameOptions"},
	{"CoverJpegQuality", "CoverArtQuality"},
	{"SimilarSongsMatchThreshold", "Matcher.FuzzyThreshold"},
	{"EnableTranscodingCancellation", "Transcoding.EnableCancellation"},
}

var removedOptions = []string{"Spotify.ID", "Spotify.Secret"}

func logDeprecatedOptions(oldName, newName string) {
	envVar := envVarName(oldName)
	newEnvVar := envVarName(newName)
	logWarning := func(oldName, newName string) {
		if newName != "" {
			log.Warn(fmt.Sprintf("Option '%s' is deprecated and will be ignored in a future release. Please use the new '%s'", oldName, newName))
		} else {
			log.Warn(fmt.Sprintf("Option '%s' is deprecated and will be ignored in a future release", oldName))
		}
	}
	if os.Getenv(envVar) != "" {
		logWarning(envVar, newEnvVar)
	}
	if viper.InConfig(oldName) {
		logWarning(oldName, newName)
	}
}

// logRemovedOptions checks if the option is set, and if yes, outputs a warning message saying the option is
// not available anymore
func logRemovedOptions(options ...string) {
	for _, option := range options {
		envVar := envVarName(option)
		logWarning := func(option string) {
			log.Warn(fmt.Sprintf("Option '%s' is not available anymore and will be ignored. Please remove it from your config", option))
		}
		if viper.InConfig(option) {
			logWarning(option)
		}
		if os.Getenv(envVar) != "" {
			logWarning(envVar)
		}
	}
}

// remapEnvVarKeysFromConfig detects ND_-prefixed keys in the config file (users mistakenly
// using environment variable names) and remaps them to canonical Viper keys with a warning.
func remapEnvVarKeysFromConfig() {
	for _, key := range viper.AllKeys() {
		if !strings.HasPrefix(key, "nd_") || !viper.InConfig(key) {
			continue
		}
		stripped := strings.TrimPrefix(key, "nd_")
		canonicalKey := ndKeyToCanonical(key)
		displayNDKey := "ND_" + strings.ToUpper(stripped)
		canonicalName := canonicalOptionName(canonicalKey)

		if viper.InConfig(canonicalKey) {
			logFatal(fmt.Sprintf(
				"Config file contains both '%s' and '%s'. Remove the ND_-prefixed version. "+
					"The 'ND_' prefix is only needed for environment variables, not config file keys.",
				displayNDKey, cmp.Or(canonicalName, toPascalCase(canonicalKey)),
			))
			return
		}

		viper.Set(canonicalKey, viper.Get(key))
		// Unknown keys get no advice here, logUnknownOptions reports them instead
		if canonicalName != "" {
			_, _ = fmt.Fprintf(os.Stderr, "WARNING: Config key '%s' uses environment variable naming. Use '%s' instead. "+
				"The 'ND_' prefix is only needed for environment variables.\n",
				displayNDKey, canonicalName,
			)
		}
	}
}

// mapDeprecatedOption is used to provide backwards compatibility for deprecated options. It should be called after
// the config has been read by viper, but before unmarshalling it into the Config struct.
func mapDeprecatedOption(legacyName, newName string) {
	// viper.Set outranks the config file, so an explicit replacement must win over the legacy value
	if viper.IsSet(legacyName) && !explicitlySet(newName) {
		viper.Set(newName, viper.Get(legacyName))
	}
}

// explicitlySet reports whether the user provided the option, ignoring defaults,
// which viper.IsSet counts as set. The ND_ spelling is also accepted in the config
// file, and remapEnvVarKeysFromConfig has already moved it out of InConfig's reach.
func explicitlySet(name string) bool {
	envVar := envVarName(name)
	return viper.InConfig(name) || os.Getenv(envVar) != "" || viper.InConfig(strings.ToLower(envVar))
}

func envVarName(option string) string {
	if option == "" {
		return ""
	}
	return "ND_" + strings.ToUpper(strings.ReplaceAll(option, ".", "_"))
}

func logUnknownOptions() {
	for _, key := range unknownConfigKeys() {
		msg := fmt.Sprintf("Option '%s' is not recognized and will be ignored", key)
		if matches := suggestOptions(key); len(matches) > 0 {
			msg += fmt.Sprintf(". Did you mean '%s'?", strings.Join(matches, "' or '"))
		}
		log.Warn(msg)
	}
}

// suggestOptions returns the known options sharing the last segment with key,
// catching options written outside their section.
func suggestOptions(key string) []string {
	key = strings.ToLower(key)
	leaf := leafKey(key)
	canonical, _ := configKeys()
	var matches []string
	for known, name := range canonical {
		// Removed options are known only so they get their own warning, never suggest them
		if known != key && leafKey(known) == leaf && !slices.Contains(removedOptions, name) {
			matches = append(matches, name)
		}
	}
	slices.Sort(matches)
	return matches
}

func leafKey(key string) string {
	return key[strings.LastIndex(key, ".")+1:]
}

// unknownConfigKeys returns config file keys that don't match any known option, so
// typos and options written outside their section don't fail silently.
func unknownConfigKeys() []string {
	// INI files keep the original [default] section alongside the merged one
	skipDefault := strings.EqualFold(filepath.Ext(viper.ConfigFileUsed()), ".ini")

	var unknown []string
	for _, key := range viper.AllKeys() {
		if !viper.InConfig(key) || canonicalOptionName(key) != "" {
			continue
		}
		if skipDefault && strings.HasPrefix(key, "default.") {
			continue
		}
		// Only ND_-prefixed keys that remapEnvVarKeysFromConfig could resolve are valid
		if strings.HasPrefix(key, "nd_") && canonicalOptionName(ndKeyToCanonical(key)) != "" {
			continue
		}
		unknown = append(unknown, key)
	}
	slices.Sort(unknown)
	return asWrittenInConfigFile(unknown)
}

func ndKeyToCanonical(key string) string {
	return strings.ReplaceAll(strings.TrimPrefix(key, "nd_"), "_", ".")
}

// canonicalOptionName returns the documented spelling of a known option key, or ""
// if it matches no option. Subkeys of free-form maps have no fixed spelling.
func canonicalOptionName(key string) string {
	keys, prefixes := configKeys()
	if name, ok := keys[key]; ok {
		return name
	}
	if slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(key, p) }) {
		return toPascalCase(key)
	}
	return ""
}

// asWrittenInConfigFile restores the casing the keys have in the config file, as
// viper lowercases every key it loads.
func asWrittenInConfigFile(keys []string) []string {
	if len(keys) == 0 {
		return nil
	}
	data, err := os.ReadFile(viper.ConfigFileUsed())
	if err != nil {
		return keys
	}
	casing := map[string]string{}
	for _, match := range configFileKeyRx.FindAllStringSubmatch(string(data), -1) {
		for segment := range strings.SplitSeq(match[1], ".") {
			lower := strings.ToLower(segment)
			casing[lower] = cmp.Or(casing[lower], segment)
		}
	}
	return slice.Map(keys, func(key string) string {
		segments := strings.Split(key, ".")
		for i, s := range segments {
			segments[i] = cmp.Or(casing[s], s)
		}
		return strings.Join(segments, ".")
	})
}

// Matches keys and section headers in all supported config formats.
var configFileKeyRx = regexp.MustCompile(`(?m)^\s*\[?\s*"?([\w.]+)"?\s*[]=:]`)

// configKeys maps every accepted option name, lowercased, to its canonical spelling,
// plus the prefixes of free-form map options (Tags, DevLogLevels).
var configKeys = sync.OnceValues(func() (map[string]string, []string) {
	keys := map[string]string{}
	var prefixes []string

	var collect func(t reflect.Type, prefix string)
	collect = func(t reflect.Type, prefix string) {
		for field := range t.Fields() {
			// `conf:"-"` marks values computed during Load, not settable in the config
			if !field.IsExported() || field.Tag.Get("conf") == "-" {
				continue
			}
			name := prefix + field.Name
			if field.Type.Kind() == reflect.Struct && !reflect.PointerTo(field.Type).Implements(textUnmarshalerType) {
				collect(field.Type, name+".")
				continue
			}
			lower := strings.ToLower(name)
			keys[lower] = name
			if field.Type.Kind() == reflect.Map {
				prefixes = append(prefixes, lower+".")
			}
		}
	}
	collect(reflect.TypeFor[configOptions](), "")

	for _, o := range deprecatedOptions {
		keys[strings.ToLower(o.name)] = o.name
	}
	for _, o := range removedOptions {
		keys[strings.ToLower(o)] = o
	}
	return keys, prefixes
})

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

// parseIniFileConfiguration is used to parse the config file when it is in INI format. For INI files, it
// would require a nested structure, so instead we unmarshal it to a map and then merge the nested [default]
// section into the root level.
func parseIniFileConfiguration() {
	cfgFile := viper.ConfigFileUsed()
	if strings.ToLower(filepath.Ext(cfgFile)) == ".ini" {
		var iniConfig map[string]any
		err := viper.Unmarshal(&iniConfig)
		if err != nil {
			logFatal("Error parsing config:", err)
		}
		cfg, ok := iniConfig["default"].(map[string]any)
		if !ok {
			logFatal("Error parsing config: missing [default] section:", iniConfig)
		}
		err = viper.MergeConfigMap(cfg)
		if err != nil {
			logFatal("Error parsing config:", err)
		}
	}
}

func disableExternalServices() {
	log.Info("All external integrations are DISABLED!")
	Server.EnableInsightsCollector = false
	Server.EnableM3UExternalAlbumArt = false
	Server.LastFM.Enabled = false
	Server.Deezer.Enabled = false
	Server.ListenBrainz.Enabled = false
	Server.Agents = ""
	if Server.UILoginBackgroundURL == consts.DefaultUILoginBackgroundURL {
		Server.UILoginBackgroundURL = consts.DefaultUILoginBackgroundURLOffline
	}
}

func validatePlaylistsPath() error {
	for path := range strings.SplitSeq(Server.PlaylistsPath, string(filepath.ListSeparator)) {
		_, err := doublestar.Match(path, "")
		if err != nil {
			return fmt.Errorf("invalid PlaylistsPath %q: %w", path, err)
		}
	}
	return nil
}

// parseLanguages parses a comma-separated language string into a slice.
// It trims whitespace from each entry and ensures at least [DefaultInfoLanguage] is returned.
func parseLanguages(lang string) []string {
	var languages []string
	for l := range strings.SplitSeq(lang, ",") {
		l = strings.TrimSpace(l)
		if l != "" {
			languages = append(languages, l)
		}
	}
	if len(languages) == 0 {
		return []string{consts.DefaultInfoLanguage}
	}
	return languages
}

func validatePurgeMissingOption() error {
	allowedValues := []string{consts.PurgeMissingNever, consts.PurgeMissingAlways, consts.PurgeMissingFull}
	valid := slices.Contains(allowedValues, Server.Scanner.PurgeMissing)
	if !valid {
		err := fmt.Errorf("invalid Scanner.PurgeMissing value: '%s'. Must be one of: %v", Server.Scanner.PurgeMissing, allowedValues)
		Server.Scanner.PurgeMissing = consts.PurgeMissingNever
		return err
	}
	return nil
}

func validateByteSize(name, value string) func() error {
	return func() error {
		size, err := humanize.ParseBytes(value)
		if err != nil {
			return fmt.Errorf("invalid %s %q: use values like '10MB', '1GB', or raw bytes like '10485760': %w", name, value, err)
		}
		if size == 0 {
			return fmt.Errorf("invalid %s %q: must be greater than zero", name, value)
		}
		if size > math.MaxInt64 {
			return fmt.Errorf("invalid %s %q: value is too large", name, value)
		}
		return nil
	}
}

func validateEnforceNonRootUser() error {
	if !Server.EnforceNonRootUser || currentGOOS() == "windows" {
		return nil
	}

	if getEUID() == 0 {
		return fmt.Errorf("EnforceNonRootUser is enabled but Navidrome is running as root")
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
	if Server.Backup.Path.String() == "" || Server.Backup.Schedule == "" || Server.Backup.Count == 0 {
		Server.Backup.Schedule = ""
		return nil
	}
	var err error
	Server.Backup.Schedule, err = validateSchedule(Server.Backup.Schedule, "Backup.Schedule")
	return err
}

func validateSchedule(schedule, field string) (string, error) {
	_, err := scheduler.ParseCrontab(schedule)
	if err != nil {
		return schedule, fmt.Errorf("invalid %s %q (see https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format): %w", field, schedule, err)
	}
	return schedule, nil
}

// validateURL checks if the provided URL is valid and has either http or https scheme.
// It returns a function that can be used as a hook to validate URLs in the config.
func validateURL(optionName, optionURL string) func() error {
	return func() error {
		if optionURL == "" {
			return nil
		}
		u, err := url.Parse(optionURL)
		if err != nil {
			return fmt.Errorf("invalid %s %q: %w", optionName, optionURL, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("invalid scheme for %s: '%s'. Only 'http' and 'https' are allowed", optionName, u.Scheme)
		}
		if u.Host == "" || u.Opaque != "" {
			return fmt.Errorf("invalid %s: '%s'. A full http(s) URL with a non-empty host is required", optionName, optionURL)
		}
		return nil
	}
}

func normalizeSearchBackend(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "fts", "legacy":
		return v
	default:
		log.Error("Invalid Search.Backend value, falling back to 'fts'", "value", value)
		return "fts"
	}
}

// toPascalCase converts a dotted lowercase config key to PascalCase for display.
// Example: "scanner.schedule" → "Scanner.Schedule"
func toPascalCase(key string) string {
	if key == "" {
		return ""
	}
	parts := strings.Split(key, ".")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, ".")
}

// AddHook is used to register initialization code that should run as soon as the config is loaded
func AddHook(hook func()) {
	hooks = append(hooks, hook)
}

// hasNDEnvVars checks if any ND_ prefixed environment variables are set (excluding ND_CONFIGFILE)
func hasNDEnvVars() bool {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "ND_") && !strings.HasPrefix(env, "ND_CONFIGFILE=") {
			return true
		}
	}
	return false
}

func setViperDefaults() {
	viper.SetDefault("musicfolder", filepath.Join(".", "music"))
	viper.SetDefault("cachefolder", "")
	viper.SetDefault("datafolder", ".")
	viper.SetDefault("loglevel", "info")
	viper.SetDefault("logfile", "")
	viper.SetDefault("address", "0.0.0.0")
	viper.SetDefault("port", 4533)
	viper.SetDefault("unixsocketperm", "0660")
	viper.SetDefault("enforcenonrootuser", false)
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
	viper.SetDefault("enablemediafiledeletion", false)
	viper.SetDefault("enableexternalservices", true)
	viper.SetDefault("enablem3uexternalalbumart", false)
	viper.SetDefault("enablemediafilecoverart", true)
	viper.SetDefault("autotranscodedownload", false)
	viper.SetDefault("defaultdownsamplingformat", consts.DefaultDownsamplingFormat)
	viper.SetDefault("search.fullstring", false)
	viper.SetDefault("search.backend", "fts")
	viper.SetDefault("matcher.preferstarred", true)
	viper.SetDefault("matcher.fuzzythreshold", 85)
	viper.SetDefault("recentlyaddedbymodtime", false)
	viper.SetDefault("prefersorttags", false)
	viper.SetDefault("ignoredarticles", "The El La Los Las Le Les Os As O A")
	viper.SetDefault("indexgroups", "A B C D E F G H I J K L M N O P Q R S T U V W X-Z(XYZ) [Unknown]([)")
	viper.SetDefault("ffmpegpath", "")
	viper.SetDefault("mpvpath", "")
	viper.SetDefault("mpvcmdtemplate", "mpv --audio-device=%d --no-audio-display %f --input-ipc-server=%s")
	viper.SetDefault("coverartpriority", "cover.*, folder.*, front.*, embedded, external")
	viper.SetDefault("coverartquality", 75)
	viper.SetDefault("enablewebpencoding", false)
	viper.SetDefault("artistartpriority", "artist.*, album/artist.*, external")
	viper.SetDefault("artistimagefolder", "")
	viper.SetDefault("discartpriority", "disc*.*, cd*.*, cover.*, folder.*, front.*, discsubtitle, embedded")
	viper.SetDefault("lyricspriority", ".ttml,.yaml,.yml,.elrc,.lrc,.srt,.txt,embedded")
	viper.SetDefault("enablegravatar", false)
	viper.SetDefault("enablefavourites", true)
	viper.SetDefault("enablestarrating", true)
	viper.SetDefault("enableuserediting", true)
	viper.SetDefault("defaulttheme", "Dark")
	viper.SetDefault("defaultlanguage", "")
	viper.SetDefault("defaultuivolume", consts.DefaultUIVolume)
	viper.SetDefault("uisearchdebouncems", consts.DefaultUISearchDebounceMs)
	viper.SetDefault("uicoverartsize", consts.DefaultUICoverArtSize)
	viper.SetDefault("enablereplaygain", true)
	viper.SetDefault("enablecoveranimation", true)
	viper.SetDefault("enablenowplaying", true)
	viper.SetDefault("uiplaybackreportinterval", consts.DefaultUIPlaybackReportInterval)
	viper.SetDefault("enableartworkupload", true)
	viper.SetDefault("maximageuploadsize", consts.DefaultMaxImageUploadSize)
	viper.SetDefault("maximagesize", consts.DefaultMaxImageSize)
	viper.SetDefault("enablesharing", true)
	viper.SetDefault("shareurl", "")
	viper.SetDefault("defaultshareexpiration", 8760*time.Hour)
	viper.SetDefault("defaultdownloadableshare", false)
	viper.SetDefault("gatrackingid", "")
	viper.SetDefault("enableinsightscollector", true)
	viper.SetDefault("enablescheduleddbanalyze", true)
	viper.SetDefault("enablelogredacting", true)
	viper.SetDefault("authrequestlimit", 5)
	viper.SetDefault("authwindowlength", 20*time.Second)
	viper.SetDefault("passwordencryptionkey", "")
	viper.SetDefault("extauth.userheader", "Remote-User")
	viper.SetDefault("extauth.trustedsources", "")
	viper.SetDefault("extauth.logouturl", "")
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
	viper.SetDefault("scanner.watcherwait", consts.DefaultWatcherWait)
	viper.SetDefault("scanner.scanonstartup", true)
	viper.SetDefault("scanner.artistjoiner", consts.ArtistJoiner)
	viper.SetDefault("scanner.artistsplitexceptions", []string{})
	viper.SetDefault("scanner.genreseparators", "")
	viper.SetDefault("scanner.groupalbumreleases", false)
	viper.SetDefault("scanner.followsymlinks", true)
	viper.SetDefault("scanner.ignoredotfolders", true)
	viper.SetDefault("scanner.purgemissing", consts.PurgeMissingNever)
	viper.SetDefault("subsonic.appendsubtitle", true)
	viper.SetDefault("subsonic.appendalbumversion", true)
	viper.SetDefault("subsonic.artistparticipations", false)
	viper.SetDefault("subsonic.defaultreportrealpath", false)
	viper.SetDefault("subsonic.enableaveragerating", true)
	viper.SetDefault("subsonic.legacyclients", "DSub")
	viper.SetDefault("subsonic.minimalclients", "SubMusic")
	viper.SetDefault("transcoding.maxconcurrent", 0)
	viper.SetDefault("transcoding.maxconcurrentperuser", 0)
	viper.SetDefault("transcoding.enablecancellation", false)
	viper.SetDefault("agents", "deezer,lastfm,listenbrainz")
	viper.SetDefault("lastfm.enabled", true)
	viper.SetDefault("lastfm.language", consts.DefaultInfoLanguage)
	viper.SetDefault("lastfm.apikey", "")
	viper.SetDefault("lastfm.secret", "")
	viper.SetDefault("lastfm.scrobblefirstartistonly", false)
	viper.SetDefault("deezer.enabled", true)
	viper.SetDefault("deezer.language", consts.DefaultInfoLanguage)
	viper.SetDefault("listenbrainz.enabled", true)
	viper.SetDefault("listenbrainz.baseurl", consts.DefaultListenBrainzBaseURL)
	viper.SetDefault("listenbrainz.artistalgorithm", consts.DefaultListenBrainzArtistAlgorithm)
	viper.SetDefault("listenbrainz.trackalgorithm", consts.DefaultListenBrainzTrackAlgorithm)
	viper.SetDefault("jellyfin.enabled", false)
	viper.SetDefault("jellyfin.servername", "")
	viper.SetDefault("enablescrobblehistory", true)
	viper.SetDefault("httpheaders.frameoptions", "DENY")
	viper.SetDefault("backup.path", "")
	viper.SetDefault("backup.schedule", "")
	viper.SetDefault("backup.count", 0)
	viper.SetDefault("pid.track", consts.DefaultTrackPID)
	viper.SetDefault("pid.album", consts.DefaultAlbumPID)
	viper.SetDefault("inspect.enabled", true)
	viper.SetDefault("inspect.maxrequests", 1)
	viper.SetDefault("inspect.backloglimit", consts.RequestThrottleBacklogLimit)
	viper.SetDefault("inspect.backlogtimeout", consts.RequestThrottleBacklogTimeout)
	viper.SetDefault("plugins.folder", "")
	viper.SetDefault("plugins.enabled", true)
	viper.SetDefault("plugins.cachesize", "200MB")
	viper.SetDefault("plugins.autoreload", false)
	viper.SetDefault("plugins.loglevel", "")

	// DevFlags. These are used to enable/disable debugging and incomplete features
	viper.SetDefault("devlogsourceline", false)
	viper.SetDefault("devenableprofiler", false)
	viper.SetDefault("devautocreateadminpassword", "")
	viper.SetDefault("devautologinusername", "")
	viper.SetDefault("devactivitypanel", true)
	viper.SetDefault("devactivitypanelupdaterate", 300*time.Millisecond)
	viper.SetDefault("devsidebarplaylists", true)
	viper.SetDefault("devshowartistpage", true)
	viper.SetDefault("devuishowconfig", true)
	viper.SetDefault("devneweventstream", true)
	viper.SetDefault("devoffsetoptimize", 50000)
	// Half the pool: streams may take up to this many connections, leaving the rest for the scanner,
	// scrobbles and the UI. See MaxOpenConns.
	viper.SetDefault("jellyfin.maxconcurrentstreams", max(2, MaxOpenConns()/2))
	viper.SetDefault("devartworkmaxrequests", max(2, runtime.NumCPU()/2))
	viper.SetDefault("devartworkthrottlebackloglimit", consts.RequestThrottleBacklogLimit)
	viper.SetDefault("devartworkthrottlebacklogtimeout", consts.RequestThrottleBacklogTimeout)
	viper.SetDefault("devartworkthrottlebuffered", true)
	// Half the CPU count (min 2), so local resolution scales with the host but stays under the
	// SQLite pool (MaxOpenConns) — leaving connections for the scanner, scrobbles and the UI.
	viper.SetDefault("devartworkworkerconcurrency", max(2, runtime.NumCPU()/2))
	// External RPS gates outbound calls to third-party services (per service); it is bounded by
	// their tolerance, not the host, so it stays a small constant regardless of CPU count.
	viper.SetDefault("devartworkexternalmaxrps", 2)
	viper.SetDefault("devartistinfotimetolive", consts.ArtistInfoTimeToLive)
	viper.SetDefault("devalbuminfotimetolive", consts.AlbumInfoTimeToLive)
	viper.SetDefault("devexternalscanner", true)
	viper.SetDefault("devscannerthreads", 5)
	viper.SetDefault("devselectivewatcher", true)
	viper.SetDefault("devinsightsinitialdelay", consts.InsightsInitialDelay)
	viper.SetDefault("devenableplayerinsights", true)
	viper.SetDefault("devenablepluginsinsights", true)
	viper.SetDefault("devplugincompilationtimeout", time.Minute)
	viper.SetDefault("devexternalartistfetchmultiplier", 1.5)
	viper.SetDefault("devpreserveunicodeinexternalcalls", false)
	viper.SetDefault("devenablemediafileprobe", true)
}

func init() {
	setViperDefaults()
}

func InitConfig(cfgFile string, loadEnvVars bool) {
	codecRegistry := viper.NewCodecRegistry()
	_ = codecRegistry.RegisterCodec("ini", ini.Codec{
		LoadOptions: ini.LoadOptions{
			UnescapeValueDoubleQuotes:   true,
			UnescapeValueCommentSymbols: true,
		},
	})
	viper.SetOptions(viper.WithCodecRegistry(codecRegistry))

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
	if loadEnvVars {
		viper.SetEnvPrefix("ND")
		replacer := strings.NewReplacer(".", "_")
		viper.SetEnvKeyReplacer(replacer)
		viper.AutomaticEnv()
	}

	err := viper.ReadInConfig()
	if viper.ConfigFileUsed() != "" && err != nil {
		logFatal("Navidrome could not open config file:", err)
	}
}

// getConfigFile returns the path to the config file, either from the flag or from the environment variable.
// If it is defined in the environment variable, it will check if the file exists.
func getConfigFile(cfgFile string) string {
	if cfgFile != "" {
		return cfgFile
	}
	cfgFile = os.Getenv("ND_CONFIGFILE")
	if cfgFile != "" {
		if _, err := os.Stat(cfgFile); err == nil { //nolint:gosec
			return cfgFile
		}
	}
	return ""
}

// MaxOpenConns is the size of the shared SQLite connection pool, used by every subsystem (scanner,
// Subsonic, Jellyfin, native API, UI).
//
// It bounds concurrent *readers*: SQLite serializes writers on a single database-wide write lock, so
// more connections buy no write parallelism. A connection is held while blocked on disk I/O or on a
// slow HTTP client, neither of which is CPU-bound — the CPU-bound knob is DevScannerThreads — so the
// count is only loosely related to core count, and the floor is what matters on small machines.
func MaxOpenConns() int {
	return max(4, runtime.NumCPU())
}
