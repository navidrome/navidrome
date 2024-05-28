package scanner2

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("processMissingTracks", func() {
	var (
		ctx context.Context
		ds  model.DataStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		ds = &tests.MockDataStore{}
	})

	It("should move the matched track when the missing track is the exact same", func() {
		missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
		matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "dir2/path2.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}

		ds.MediaFile(ctx).Put(&missingTrack)
		ds.MediaFile(ctx).Put(&matchedTrack)

		in := &missingTracks{
			missing: []model.MediaFile{missingTrack},
			matched: []model.MediaFile{matchedTrack},
		}

		_, err := processMissingTracks(ctx, ds)(in)
		Expect(err).ToNot(HaveOccurred())

		movedTrack, _ := ds.MediaFile(ctx).Get("1")
		Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
	})

	It("should move the matched track when the missing track has the same tags and filename", func() {
		missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
		matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "path1.flac", Tags: model.Tags{"title": []string{"title1"}}, Size: 200}

		ds.MediaFile(ctx).Put(&missingTrack)
		ds.MediaFile(ctx).Put(&matchedTrack)

		in := &missingTracks{
			missing: []model.MediaFile{missingTrack},
			matched: []model.MediaFile{matchedTrack},
		}

		_, err := processMissingTracks(ctx, ds)(in)
		Expect(err).ToNot(HaveOccurred())

		movedTrack, _ := ds.MediaFile(ctx).Get("1")
		Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
		Expect(movedTrack.Size).To(Equal(matchedTrack.Size))
	})

	It("should return an error when there's an error moving the matched track", func() {
		missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}}
		matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}}

		ds.MediaFile(ctx).Put(&missingTrack)
		ds.MediaFile(ctx).Put(&matchedTrack)

		in := &missingTracks{
			missing: []model.MediaFile{missingTrack},
			matched: []model.MediaFile{matchedTrack},
		}

		// Simulate an error when moving the matched track
		ds.MediaFile(ctx).Delete("2")

		_, err := processMissingTracks(ctx, ds)(in)
		Expect(err).To(HaveOccurred())
	})
})
