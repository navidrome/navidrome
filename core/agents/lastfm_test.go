package agents

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("lastfmAgent", func() {
	Describe("lastFMConstructor", func() {
		It("uses default api key and language if not configured", func() {
			conf.Server.LastFM.ApiKey = ""
			agent := lastFMConstructor(context.TODO())
			Expect(agent.(*lastfmAgent).apiKey).To(Equal(lastFMAPIKey))
			Expect(agent.(*lastfmAgent).lang).To(Equal("en"))
		})

		It("uses configured api key and language", func() {
			conf.Server.LastFM.ApiKey = "123"
			conf.Server.LastFM.Language = "pt"
			agent := lastFMConstructor(context.TODO())
			Expect(agent.(*lastfmAgent).apiKey).To(Equal("123"))
			Expect(agent.(*lastfmAgent).lang).To(Equal("pt"))
		})
	})
})
