package conf_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/navidrome/navidrome/conf"
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
})
