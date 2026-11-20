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
	levels []int16
	err    error
	calls  int
}

func (f *fakeWaveform) Get(context.Context, *model.MediaFile) ([]int16, error) {
	f.calls++
	return f.levels, f.err
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
		wf = &fakeWaveform{levels: []int16{waveform.Silence, -3142, -602, -1204}}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, wf)
	})

	It("returns every stored level when samples is missing", func() {
		response, err := router.GetWaveform(newGetRequest("id=1"))
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Waveform).To(Equal(&responses.Waveform{
			ID: "1", Duration: 12.5, Interval: 100, Samples: 4, Levels: []float64{-120, -31.42, -6.02, -12.04},
		}))
	})

	It("downsamples to exactly samples values, keeping the highest of each group", func() {
		response, err := router.GetWaveform(newGetRequest("id=1", "samples=2"))
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Waveform).To(Equal(&responses.Waveform{
			ID: "1", Duration: 12.5, Interval: 200, Samples: 2, Levels: []float64{-31.42, -6.02},
		}))
	})

	It("returns every stored level when samples is larger than the stored count", func() {
		response, err := router.GetWaveform(newGetRequest("id=1", "samples=100000"))
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Waveform.Samples).To(Equal(int32(4)))
		Expect(response.Waveform.Interval).To(Equal(100.0))
	})

	It("rejects samples that are not a number", func() {
		_, err := router.GetWaveform(newGetRequest("id=1", "samples=abc"))
		Expect(err).To(HaveOccurred())
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
			Expect(wf.calls).To(BeZero())
		})
	})

	It("returns a Subsonic error when no waveform service is configured", func() {
		router.waveform = nil
		_, err := router.GetWaveform(newGetRequest("id=1"))
		var subErr subError
		Expect(errors.As(err, &subErr)).To(BeTrue())
	})
})
