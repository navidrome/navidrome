package scanner_test

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PIDConfChanged", func() {
	var ds *tests.MockDataStore
	ctx := context.Background()

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.PID.Album = "album_spec"
		conf.Server.PID.Track = "track_spec"
		ds = &tests.MockDataStore{}
	})

	It("reports no change when every library matches its effective specs", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "album_spec", ScannedPIDTrack: "track_spec"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeFalse())
	})

	It("reports a change when the global album spec differs from what was scanned", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "old_album", ScannedPIDTrack: "track_spec"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeTrue())
	})

	It("reports a change when only one library has an override that was never scanned", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "album_spec", ScannedPIDTrack: "track_spec"},
			{ID: 2, PIDAlbum: "folder", ScannedPIDAlbum: "album_spec", ScannedPIDTrack: "track_spec"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeTrue())
	})

	It("ignores case differences", func() {
		repo := &tests.MockLibraryRepo{}
		repo.SetData(model.Libraries{
			{ID: 1, ScannedPIDAlbum: "ALBUM_SPEC", ScannedPIDTrack: "TRACK_SPEC"},
		})
		ds.MockedLibrary = repo
		Expect(scanner.PIDConfChanged(ctx, ds)).To(BeFalse())
	})
})
