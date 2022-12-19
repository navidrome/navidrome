package core

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Artwork", func() {
	var ds model.DataStore
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "222", EmbedArtPath: "tests/fixtures/test.mp3"},
			{ID: "333"},
			{ID: "444", EmbedArtPath: "tests/fixtures/cover.jpg"},
		})
		ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "123", AlbumID: "222", Path: "tests/fixtures/test.mp3", HasCoverArt: true},
			{ID: "456", AlbumID: "222", Path: "tests/fixtures/test.ogg", HasCoverArt: false},
		})
	})

})
