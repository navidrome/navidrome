package conf_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}

var _ = Describe("Configuration", func() {
	BeforeEach(func() {
		// Reset viper configuration
		viper.Reset()
		conf.SetViperDefaults()
		viper.SetDefault("datafolder", GinkgoT().TempDir())
		viper.SetDefault("loglevel", "error")
		conf.ResetConf()

		// Panic instead of exiting on fatal errors to allow testing error conditions
		DeferCleanup(conf.SetLogFatal(func(args ...any) {
			panic(fmt.Sprint(args...))
		}))
	})

	Describe("ParseLanguages", func() {
		It("parses single language", func() {
			Expect(conf.ParseLanguages("en")).To(Equal([]string{"en"}))
		})

		It("parses multiple comma-separated languages", func() {
			Expect(conf.ParseLanguages("pt,en")).To(Equal([]string{"pt", "en"}))
		})

		It("trims whitespace from languages", func() {
			Expect(conf.ParseLanguages(" pt , en ")).To(Equal([]string{"pt", "en"}))
		})

		It("returns default 'en' when empty", func() {
			Expect(conf.ParseLanguages("")).To(Equal([]string{"en"}))
		})

		It("returns default 'en' when only whitespace", func() {
			Expect(conf.ParseLanguages("   ")).To(Equal([]string{"en"}))
		})

		It("handles multiple languages with various spacing", func() {
			Expect(conf.ParseLanguages("ja, pt, en")).To(Equal([]string{"ja", "pt", "en"}))
		})
	})

	Describe("scheduled DB analysis", func() {
		It("is enabled by default", func() {
			conf.Load(true)
			Expect(conf.Server.EnableScheduledDBAnalyze).To(BeTrue())
		})

		It("can be disabled", func() {
			viper.Set("enablescheduleddbanalyze", false)
			conf.Load(true)
			Expect(conf.Server.EnableScheduledDBAnalyze).To(BeFalse())
		})
	})

	Describe("ValidateURL", func() {
		It("accepts a valid http URL", func() {
			fn := conf.ValidateURL("TestOption", "http://example.com/path")
			Expect(fn()).To(Succeed())
		})

		It("accepts a valid https URL", func() {
			fn := conf.ValidateURL("TestOption", "https://example.com/path")
			Expect(fn()).To(Succeed())
		})

		It("rejects a URL with no scheme", func() {
			fn := conf.ValidateURL("TestOption", "example.com/path")
			Expect(fn()).To(MatchError(ContainSubstring("invalid scheme")))
		})

		It("rejects a URL with an unsupported scheme", func() {
			fn := conf.ValidateURL("TestOption", "javascript://example.com/path")
			Expect(fn()).To(MatchError(ContainSubstring("invalid scheme")))
		})

		It("accepts an empty URL (optional config)", func() {
			fn := conf.ValidateURL("TestOption", "")
			Expect(fn()).To(Succeed())
		})

		It("includes the option name in the error message", func() {
			fn := conf.ValidateURL("MyOption", "ftp://example.com")
			Expect(fn()).To(MatchError(ContainSubstring("MyOption")))
		})

		It("rejects a URL that cannot be parsed", func() {
			fn := conf.ValidateURL("TestOption", "://invalid")
			Expect(fn()).To(HaveOccurred())
		})

		It("rejects a URL without a host", func() {
			fn := conf.ValidateURL("TestOption", "http:///path")
			Expect(fn()).To(MatchError(ContainSubstring("non-empty host is required")))
		})
	})

	DescribeTable("NormalizeSearchBackend",
		func(input, expected string) {
			Expect(conf.NormalizeSearchBackend(input)).To(Equal(expected))
		},
		Entry("accepts 'fts'", "fts", "fts"),
		Entry("accepts 'legacy'", "legacy", "legacy"),
		Entry("normalizes 'FTS' to lowercase", "FTS", "fts"),
		Entry("normalizes 'Legacy' to lowercase", "Legacy", "legacy"),
		Entry("trims whitespace", "  fts  ", "fts"),
		Entry("falls back to 'fts' for 'fts5'", "fts5", "fts"),
		Entry("falls back to 'fts' for unrecognized values", "invalid", "fts"),
		Entry("falls back to 'fts' for empty string", "", "fts"),
	)

	DescribeTable("ToPascalCase",
		func(input, expected string) {
			Expect(conf.ToPascalCase(input)).To(Equal(expected))
		},
		Entry("simple key", "address", "Address"),
		Entry("dotted key", "scanner.schedule", "Scanner.Schedule"),
		Entry("already capitalized", "Address", "Address"),
		Entry("multi-segment", "lastfm.enabled", "Lastfm.Enabled"),
		Entry("empty string", "", ""),
	)

	Describe("remapEnvVarKeysFromConfig", func() {
		BeforeEach(func() {
			viper.Reset()
			conf.SetViperDefaults()
			viper.SetDefault("datafolder", GinkgoT().TempDir())
			viper.SetDefault("loglevel", "error")
			conf.ResetConf()
		})

		It("remaps ND_-prefixed keys to canonical keys", func() {
			filename := filepath.Join("testdata", "cfg_nd_keys.toml")
			conf.InitConfig(filename, false)
			conf.Load(true)

			Expect(conf.Server.Address).To(Equal("127.0.0.1"))
			Expect(conf.Server.Port).To(Equal(4531))
			Expect(conf.Server.Scanner.Schedule).To(Equal("@every 1h"))
		})

		It("exits with fatal error when both ND_ and canonical key exist", func() {
			filename := filepath.Join("testdata", "cfg_nd_conflict.toml")
			conf.InitConfig(filename, false)

			Expect(func() { conf.Load(true) }).To(PanicWith(And(
				ContainSubstring("ND_ADDRESS"),
				ContainSubstring("Address"),
				ContainSubstring("only needed for environment variables"),
			)))
		})

		It("does nothing when no ND_ keys are present", func() {
			filename := filepath.Join("testdata", "cfg.toml")
			conf.InitConfig(filename, false)
			conf.Load(true)

			// Verify normal config loading still works
			Expect(conf.Server.MusicFolder).To(Equal("/toml/music"))
		})
	})

	Describe("unknownConfigKeys", func() {
		BeforeEach(func() {
			viper.Reset()
			conf.SetViperDefaults()
			viper.SetDefault("datafolder", GinkgoT().TempDir())
			viper.SetDefault("loglevel", "error")
			conf.ResetConf()
		})

		It("reports misplaced and misspelled options, as spelled in the config file", func() {
			conf.InitConfig(filepath.Join("testdata", "cfg_unknown_keys.toml"), false)
			conf.Load(true)

			Expect(conf.UnknownConfigKeys()).To(ConsistOf(
				"ArtistSplitExceptions", "EnableDownlods", "Whatever.Foo",
			))
		})

		DescribeTable("recovers the original casing in all supported formats",
			func(file string) {
				conf.InitConfig(filepath.Join("testdata", file), false)
				conf.Load(true)

				Expect(conf.UnknownConfigKeys()).To(ConsistOf("NotAnOption"))
			},
			Entry("TOML", "cfg_unknown_casing.toml"),
			Entry("YAML", "cfg_unknown_casing.yaml"),
			Entry("JSON", "cfg_unknown_casing.json"),
			Entry("INI", "cfg_unknown_casing.ini"),
		)

		It("does not report valid, deprecated or free-form keys", func() {
			conf.InitConfig(filepath.Join("testdata", "cfg.toml"), false)
			conf.Load(true)

			Expect(conf.UnknownConfigKeys()).To(BeEmpty())
		})

		It("does not report the [default] section of INI files", func() {
			conf.InitConfig(filepath.Join("testdata", "cfg.ini"), false)
			conf.Load(true)

			Expect(conf.UnknownConfigKeys()).To(BeEmpty())
		})

		DescribeTable("SuggestOptions",
			func(key string, expected []string) {
				Expect(conf.SuggestOptions(key)).To(Equal(expected))
			},
			Entry("suggests the section of a misplaced option", "artistsplitexceptions",
				[]string{"Scanner.ArtistSplitExceptions"}),
			Entry("suggests the section of a misplaced nested option", "backup.fuzzythreshold",
				[]string{"Matcher.FuzzyThreshold"}),
			Entry("suggests every section defining the option", "schedule",
				[]string{"Backup.Schedule", "Scanner.Schedule"}),
			Entry("suggests nothing for a typo", "enabledownlods", nil),
		)

		It("does not report ND_-prefixed keys, as they are remapped", func() {
			conf.InitConfig(filepath.Join("testdata", "cfg_nd_keys.toml"), false)
			conf.Load(true)

			Expect(conf.UnknownConfigKeys()).To(BeEmpty())
		})

		It("reports ND_-prefixed keys that remap to no known option", func() {
			conf.InitConfig(filepath.Join("testdata", "cfg_nd_bogus.toml"), false)
			conf.Load(true)

			Expect(conf.UnknownConfigKeys()).To(ConsistOf("ND_TOTALLY_BOGUS_OPTION"))
			Expect(conf.Server.Scanner.Schedule).To(Equal("@every 1h"))
		})

		It("migrates every deprecated option that has a replacement", func() {
			conf.InitConfig(filepath.Join("testdata", "cfg_deprecated_search.toml"), false)
			conf.Load(true)

			Expect(conf.Server.Search.FullString).To(BeTrue())
			Expect(conf.UnknownConfigKeys()).To(BeEmpty())
		})

		It("warns about each unrecognized option at startup", func() {
			var logBuf bytes.Buffer
			log.SetOutput(&logBuf)
			DeferCleanup(func() { log.SetOutput(GinkgoWriter) })

			conf.InitConfig(filepath.Join("testdata", "cfg_warning_output.toml"), false)
			conf.Load(true)

			Expect(logBuf.String()).To(ContainSubstring(
				"Option 'ArtistSplitExceptions' is not recognized and will be ignored. " +
					"Did you mean 'Scanner.ArtistSplitExceptions'?"))
			Expect(logBuf.String()).To(ContainSubstring(
				"Option 'EnableDownlods' is not recognized and will be ignored"))
			Expect(logBuf.String()).ToNot(ContainSubstring("ArtistJoiner"))
		})

		Context("with runtime-computed and removed options in the config", func() {
			BeforeEach(func() {
				conf.InitConfig(filepath.Join("testdata", "cfg_runtime_fields.toml"), false)
				conf.Load(true)
			})

			It("reports values computed during Load, which the config cannot set", func() {
				Expect(conf.UnknownConfigKeys()).To(ContainElements("ConfigFile", "LastFM.Languages"))
			})

			It("never suggests a removed option", func() {
				Expect(conf.SuggestOptions("id")).To(BeEmpty())
			})

			It("keeps an explicit replacement over the deprecated value", func() {
				Expect(conf.Server.Search.FullString).To(BeFalse())
			})
		})
	})

	Describe("logFatal", func() {
		var invalidPath string
		BeforeEach(func() {
			viper.Reset()
			conf.SetViperDefaults()
			viper.SetDefault("loglevel", "error")
			conf.ResetConf()

			// Create a file so that any path under it is invalid on all OSes
			f, err := os.CreateTemp(GinkgoT().TempDir(), "blocker")
			Expect(err).ToNot(HaveOccurred())
			f.Close()
			invalidPath = filepath.Join(f.Name(), "subdir")
		})

		It("is called when LoadFromFile gets an invalid config file", func() {
			Expect(func() {
				conf.LoadFromFile(filepath.Join(invalidPath, "file.toml"))
			}).To(PanicWith(ContainSubstring("Error reading config file")))
		})

		It("is called when LogFile path is not writable", func() {
			viper.SetDefault("datafolder", GinkgoT().TempDir())
			viper.SetDefault("logfile", filepath.Join(invalidPath, "log.txt"))
			Expect(func() {
				conf.Load(true)
			}).To(PanicWith(ContainSubstring("Error creating log file directory")))
		})

		It("is called when BaseURL is invalid", func() {
			viper.SetDefault("datafolder", GinkgoT().TempDir())
			viper.SetDefault("baseurl", "://invalid")
			Expect(func() {
				conf.Load(true)
			}).To(PanicWith(ContainSubstring("Invalid BaseURL")))
		})

	})

	Describe("ValidateByteSize", func() {
		DescribeTable("accepts valid size values",
			func(input string) {
				Expect(conf.ValidateByteSize("MaxImageSize", input)()).To(Succeed())
			},
			Entry("megabytes", "10MB"),
			Entry("gigabytes", "1GB"),
			Entry("raw bytes", "10485760"),
			Entry("mebibytes", "10MiB"),
			Entry("lower case", "50mb"),
		)

		DescribeTable("rejects invalid size values",
			func(input string) {
				Expect(conf.ValidateByteSize("MaxImageSize", input)()).To(MatchError(ContainSubstring("invalid MaxImageSize")))
			},
			Entry("garbage string", "not-a-size"),
			Entry("negative-looking", "-10MB"),
			Entry("zero", "0"),
			Entry("zero with unit", "0MB"),
			Entry("overflows int64", "9223372036854775808"),
		)
	})

	Describe("MaxImageSize floor", func() {
		BeforeEach(func() {
			viper.Reset()
			conf.SetViperDefaults()
			viper.SetDefault("datafolder", GinkgoT().TempDir())
			viper.SetDefault("loglevel", "error")
			conf.ResetConf()
		})

		It("is raised to MaxImageUploadSize when configured lower", func() {
			viper.SetDefault("maximagesize", "5MB")
			viper.SetDefault("maximageuploadsize", "50MB")
			conf.Load(true)
			Expect(conf.Server.MaxImageSize).To(Equal("50MB"))
		})

		It("keeps a larger MaxImageSize unchanged", func() {
			viper.SetDefault("maximagesize", "30MB")
			conf.Load(true)
			Expect(conf.Server.MaxImageSize).To(Equal("30MB"))
		})
	})

	Describe("EnforceNonRootUser", func() {
		It("defaults to false", func() {
			conf.Load(true)

			Expect(conf.Server.EnforceNonRootUser).To(BeFalse())
		})

		It("allows startup for non-root users when enabled", func() {
			DeferCleanup(conf.SetRuntimeInfoForTest("linux", 1000))
			viper.Set("enforcenonrootuser", true)

			conf.Load(true)

			Expect(conf.Server.EnforceNonRootUser).To(BeTrue())
		})

		It("exits when enabled and running as root without having created a data folder", func() {
			// Create a path that doesn't exist yet
			tempBase := GinkgoT().TempDir()
			nonExistentDataFolder := filepath.Join(tempBase, "nonexistent", "data")
			DeferCleanup(conf.SetRuntimeInfoForTest("linux", 0))
			viper.Set("enforcenonrootuser", true)
			viper.Set("datafolder", nonExistentDataFolder)

			// Attempt to load config as root user - should fail before creating directories
			Expect(func() {
				conf.Load(true)
			}).To(PanicWith(ContainSubstring("EnforceNonRootUser is enabled but Navidrome is running as root")))

			// Verify that the data folder was NOT created
			Expect(nonExistentDataFolder).ToNot(BeAnExistingFile())
		})

		It("is a no-op on non-unix platforms", func() {
			DeferCleanup(conf.SetRuntimeInfoForTest("windows", 0))
			viper.Set("enforcenonrootuser", true)

			conf.Load(true)

			Expect(conf.Server.EnforceNonRootUser).To(BeTrue())
		})
	})

	DescribeTable("should load configuration from",
		func(format string) {
			filename := filepath.Join("testdata", "cfg."+format)

			// Initialize config with the test file
			conf.InitConfig(filename, false)
			// Load the configuration (with noConfigDump=true)
			conf.Load(true)

			// Execute the format-specific assertions
			Expect(conf.Server.MusicFolder).To(Equal(fmt.Sprintf("/%s/music", format)))
			Expect(conf.Server.UIWelcomeMessage).To(Equal("Welcome " + format))
			Expect(conf.Server.Tags["custom"].Aliases).To(Equal([]string{format, "test"}))
			Expect(conf.Server.Tags["artist"].Split).To(Equal([]string{";"}))

			// Check deprecated option mapping
			Expect(conf.Server.ExtAuth.UserHeader).To(Equal("X-Auth-User"))

			// The config file used should be the one we created
			Expect(conf.Server.ConfigFile).To(Equal(filename))
		},
		Entry("TOML format", "toml"),
		Entry("YAML format", "yaml"),
		Entry("INI format", "ini"),
		Entry("JSON format", "json"),
	)

	It("should use default values for negative duration fields", func() {
		filename := filepath.Join("testdata", "invalid_duration.toml")
		conf.InitConfig(filename, false)
		conf.Load(true)

		server := conf.Server
		Expect(server.SessionTimeout).To(Equal(consts.DefaultSessionTimeout))
		Expect(server.SmartPlaylistRefreshDelay).To(Equal(consts.DefaultSmartRefresh))
		Expect(server.DefaultShareExpiration).To(Equal(consts.DefaultShareExpiration))
		Expect(server.UIPlaybackReportInterval).To(Equal(consts.DefaultUIPlaybackReportInterval))
		Expect(server.AuthWindowLength).To(Equal(consts.DefaultAuthWindowLength))
		Expect(server.Scanner.WatcherWait).To(Equal(consts.DefaultWatcherWait))

		Expect(server.DevActivityPanelUpdateRate).To(Equal(consts.DefaultActivityPanelUpdateRate))
		Expect(server.DevArtworkThrottleBacklogTimeout).To(Equal(consts.RequestThrottleBacklogTimeout))
		Expect(server.DevArtistInfoTimeToLive).To(Equal(consts.ArtistInfoTimeToLive))
		Expect(server.DevAlbumInfoTimeToLive).To(Equal(consts.AlbumInfoTimeToLive))
		Expect(server.DevInsightsInitialDelay).To(Equal(consts.InsightsInitialDelay))
		Expect(server.DevPluginCompilationTimeout).To(Equal(consts.DefaultPluginCompilationTimeout))
	})

	It("should use parsed values (0) for duration fields", func() {
		conf.InitConfig(filepath.Join("testdata", "valid_duration.toml"), false)
		conf.Load(true)

		configured := 0 * time.Second

		server := conf.Server
		Expect(server.SessionTimeout).To(Equal(configured))
		Expect(server.SmartPlaylistRefreshDelay).To(Equal(configured))
		Expect(server.DefaultShareExpiration).To(Equal(configured))
		Expect(server.UIPlaybackReportInterval).To(Equal(configured))
		Expect(server.AuthWindowLength).To(Equal(configured))
		Expect(server.Scanner.WatcherWait).To(Equal(configured))

		Expect(server.DevActivityPanelUpdateRate).To(Equal(configured))
		Expect(server.DevArtworkThrottleBacklogTimeout).To(Equal(configured))
		Expect(server.DevArtistInfoTimeToLive).To(Equal(configured))
		Expect(server.DevAlbumInfoTimeToLive).To(Equal(configured))
		Expect(server.DevInsightsInitialDelay).To(Equal(configured))
		Expect(server.DevPluginCompilationTimeout).To(Equal(configured))
	})
})
