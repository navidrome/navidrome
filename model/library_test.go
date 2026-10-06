package model_test

import (
	"encoding/json"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Library PID config", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.PID.Album = "global_album"
		conf.Server.PID.Track = "global_track"
	})

	Describe("EffectivePID", func() {
		It("falls back to the global config", func() {
			Expect(model.Library{}.EffectivePID()).To(Equal(model.PIDConfig{Track: "global_track", Album: "global_album"}))
		})
		It("uses the library overrides", func() {
			lib := model.Library{PIDAlbum: "folder", PIDTrack: "title"}
			Expect(lib.EffectivePID()).To(Equal(model.PIDConfig{Track: "title", Album: "folder"}))
		})
	})

	Describe("PIDChanged", func() {
		It("is false when the scanned specs match, ignoring case", func() {
			lib := model.Library{ScannedPIDAlbum: "GLOBAL_ALBUM", ScannedPIDTrack: "global_track"}
			Expect(lib.PIDChanged()).To(BeFalse())
		})
		It("is true when the album override differs from the scanned spec", func() {
			lib := model.Library{PIDAlbum: "folder", ScannedPIDAlbum: "global_album", ScannedPIDTrack: "global_track"}
			Expect(lib.PIDChanged()).To(BeTrue())
		})
		It("is true when only the track spec changed", func() {
			lib := model.Library{PIDTrack: "title", ScannedPIDAlbum: "global_album", ScannedPIDTrack: "global_track"}
			Expect(lib.PIDChanged()).To(BeTrue())
		})
		It("is true when the global config changed for a library without overrides", func() {
			lib := model.Library{ScannedPIDAlbum: "old_album", ScannedPIDTrack: "global_track"}
			Expect(lib.PIDChanged()).To(BeTrue())
		})
		It("is true for a library that was never scanned", func() {
			Expect(model.Library{}.PIDChanged()).To(BeTrue())
		})
	})

	Describe("NeedsPIDRescan", func() {
		It("is false for a library that never finished a scan", func() {
			Expect(model.Library{PIDAlbum: "folder"}.NeedsPIDRescan()).To(BeFalse())
		})
		It("is true for a scanned library whose PID config changed", func() {
			lib := model.Library{PIDAlbum: "folder", ScannedPIDAlbum: "global_album", ScannedPIDTrack: "global_track", LastScanAt: time.Now()}
			Expect(lib.NeedsPIDRescan()).To(BeTrue())
		})
		It("is false for a scanned library whose PID config did not change", func() {
			lib := model.Library{ScannedPIDAlbum: "global_album", ScannedPIDTrack: "global_track", LastScanAt: time.Now()}
			Expect(lib.NeedsPIDRescan()).To(BeFalse())
		})
	})

	It("does not expose the scanned specs in JSON", func() {
		data, err := json.Marshal(model.Library{PIDAlbum: "folder", ScannedPIDAlbum: "secret_album", ScannedPIDTrack: "secret_track"})
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(ContainSubstring(`"pidAlbum":"folder"`))
		Expect(string(data)).ToNot(ContainSubstring("secret_"))
	})
})
