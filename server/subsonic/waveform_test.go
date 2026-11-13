package subsonic

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/waveform"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeWaveform struct {
	peaks   []uint8
	err     error
	samples int
}

func (f *fakeWaveform) Get(_ context.Context, _ *model.MediaFile, samples int) ([]uint8, error) {
	f.samples = samples
	return waveform.Downsample(f.peaks, samples), f.err
}

var _ = Describe("GetWaveform", func() {
	var router *Router
	var wf *fakeWaveform
	var mockRepo *tests.MockMediaFileRepo

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableWaveform = true
		mockRepo = tests.CreateMockMediaFileRepo()
		mockRepo.SetData(model.MediaFiles{{ID: "1", Duration: 12.5}})
		ds := &tests.MockDataStore{MockedMediaFile: mockRepo}
		wf = &fakeWaveform{peaks: []uint8{0, 64, 255, 128}}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, wf)
	})

	It("returns the song's peaks", func() {
		response, err := router.GetWaveform(newGetRequest("id=1", "samples=4"))
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Waveform).To(Equal(&responses.Waveform{
			ID: "1", Duration: 12.5, Samples: 4, Peaks: []int32{0, 64, 255, 128},
		}))
	})

	It("uses the default resolution when samples is missing", func() {
		_, err := router.GetWaveform(newGetRequest("id=1"))
		Expect(err).ToNot(HaveOccurred())
		Expect(wf.samples).To(Equal(waveform.DefaultSamples))
	})

	It("caps samples at the stored resolution", func() {
		_, err := router.GetWaveform(newGetRequest("id=1", "samples=100000"))
		Expect(err).ToNot(HaveOccurred())
		Expect(wf.samples).To(Equal(waveform.MaxSamples))
	})

	It("rejects non-positive samples", func() {
		_, err := router.GetWaveform(newGetRequest("id=1", "samples=0"))
		Expect(err).To(HaveOccurred())
	})

	It("requires an id", func() {
		_, err := router.GetWaveform(newGetRequest())
		Expect(err).To(HaveOccurred())
	})

	It("returns not found for unknown songs", func() {
		_, err := router.GetWaveform(newGetRequest("id=2"))
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("returns a generic Subsonic error when generation fails", func() {
		wf.err = errors.New("stat /music/secret/song.flac: no such file or directory")
		_, err := router.GetWaveform(newGetRequest("id=1"))
		var subErr subError
		Expect(errors.As(err, &subErr)).To(BeTrue())
		Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		Expect(err.Error()).ToNot(ContainSubstring("/music"))
	})

	Context("when waveforms are disabled", func() {
		BeforeEach(func() {
			conf.Server.EnableWaveform = false
		})

		It("returns a Subsonic error without generating anything", func() {
			_, err := router.GetWaveform(newGetRequest("id=1"))
			var subErr subError
			Expect(errors.As(err, &subErr)).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
			Expect(wf.samples).To(BeZero())
		})
	})

	It("returns a Subsonic error when no waveform service is configured", func() {
		router.waveform = nil
		_, err := router.GetWaveform(newGetRequest("id=1"))
		var subErr subError
		Expect(errors.As(err, &subErr)).To(BeTrue())
	})
})
