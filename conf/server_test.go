package conf_test

import (
	"sync"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server state", func() {
	BeforeEach(func() {
		conf.Update(func(c *conf.Options) {
			c.Tags = map[string]conf.TagConf{"genre": {Aliases: []string{"genre", "style"}}}
			c.DevLogLevels = map[string]string{"scanner": "info"}
			c.Scanner.ArtistSplitExceptions = []string{"AC/DC"}
		})
	})

	It("keeps a copy taken from Server unchanged by later updates", func() {
		before := conf.Server()

		conf.Update(func(c *conf.Options) {
			c.Tags["genre"].Aliases[0] = "changed"
			c.DevLogLevels["scanner"] = "debug"
			c.Scanner.ArtistSplitExceptions[0] = "changed"
		})

		Expect(before.Tags["genre"].Aliases).To(Equal([]string{"genre", "style"}))
		Expect(before.DevLogLevels).To(Equal(map[string]string{"scanner": "info"}))
		Expect(before.Scanner.ArtistSplitExceptions).To(Equal([]string{"AC/DC"}))
		Expect(conf.Server().DevLogLevels["scanner"]).To(Equal("debug"))
	})

	It("restores map and slice contents changed in place by Update", func() {
		restore := conf.SnapshotConfig()
		conf.Update(func(c *conf.Options) {
			c.Tags["genre"].Aliases[0] = "changed"
			c.DevLogLevels["scanner"] = "debug"
		})

		restore()

		Expect(conf.Server().Tags["genre"].Aliases).To(Equal([]string{"genre", "style"}))
		Expect(conf.Server().DevLogLevels).To(Equal(map[string]string{"scanner": "info"}))
	})

	It("lets readers run while the config is updated and restored", func() {
		stop := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					_ = conf.Server().DevLogLevels["scanner"]
					_ = conf.Server().Scanner.Enabled
				}
			}
		})

		for range 100 {
			restore := conf.SnapshotConfig()
			conf.Update(func(c *conf.Options) {
				c.DevLogLevels["scanner"] = "debug"
				c.Scanner.Enabled = !c.Scanner.Enabled
			})
			restore()
		}
		close(stop)
		wg.Wait()
	})

	It("keeps every change when updates run at the same time", func() {
		conf.Update(func(c *conf.Options) { c.Port = 0 })
		var wg sync.WaitGroup
		for range 50 {
			wg.Go(func() {
				for range 20 {
					conf.Update(func(c *conf.Options) { c.Port++ })
				}
			})
		}
		wg.Wait()

		Expect(conf.Server().Port).To(Equal(1000))
	})
})
