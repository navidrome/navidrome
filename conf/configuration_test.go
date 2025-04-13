package conf_test

import (
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/navidrome/navidrome/conf"
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
		viper.SetDefault("datafolder", GinkgoT().TempDir())
		viper.SetDefault("loglevel", "error")
		ResetConf()
	})

	DescribeTable("should load configuration from",
		func(format string) {
			filename := filepath.Join("testdata", "cfg."+format)

			// Initialize config with the test file
			InitConfig(filename)
			// Load the configuration (with noConfigDump=true)
			Load(true)

			// Execute the format-specific assertions
			Expect(Server.MusicFolder).To(Equal(fmt.Sprintf("/%s/music", format)))
			Expect(Server.UIWelcomeMessage).To(Equal("Welcome " + format))
			Expect(Server.Tags["custom"].Aliases).To(Equal([]string{format, "test"}))

			// The config file used should be the one we created
			Expect(Server.ConfigFile).To(Equal(filename))
		},
		Entry("TOML format", "toml"),
		Entry("YAML format", "yaml"),
		Entry("INI format", "ini"),
		Entry("JSON format", "json"),
	)
})
