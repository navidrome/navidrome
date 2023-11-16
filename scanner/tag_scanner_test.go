package scanner

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/path_hash"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagScanner", func() {
	Describe("loadAllAudioFiles", func() {
		It("return all audio files from the folder", func() {
			files, err := loadAllAudioFiles("tests/fixtures")
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(10))
			Expect(files).To(HaveKey("tests/fixtures/test.aiff"))
			Expect(files).To(HaveKey("tests/fixtures/test.flac"))
			Expect(files).To(HaveKey("tests/fixtures/test.mp3"))
			Expect(files).To(HaveKey("tests/fixtures/test.ogg"))
			Expect(files).To(HaveKey("tests/fixtures/test.wav"))
			Expect(files).To(HaveKey("tests/fixtures/test.wma"))
			Expect(files).To(HaveKey("tests/fixtures/test.wv"))
			Expect(files).To(HaveKey("tests/fixtures/test_no_read_permission.ogg"))
			Expect(files).To(HaveKey("tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
			Expect(files).To(HaveKey("tests/fixtures/01 Invisible (RED) Edit Version.m4a"))
			Expect(files).ToNot(HaveKey("tests/fixtures/._02 Invisible.mp3"))
			Expect(files).ToNot(HaveKey("tests/fixtures/playlist.m3u"))
		})

		It("returns error if path does not exist", func() {
			_, err := loadAllAudioFiles("./INVALID/PATH")
			Expect(err).To(HaveOccurred())
		})

		It("returns empty map if there are no audio files in path", func() {
			Expect(loadAllAudioFiles("tests/fixtures/empty_folder")).To(BeEmpty())
		})
	})

	Describe("TagScanner", func() {
		var ds model.DataStore
		var pls core.Playlists
		var cw artwork.CacheWarmer
		var ctx context.Context

		BeforeEach(func() {
			ds = &tests.MockDataStore{}
			pls = core.NewPlaylists(ds)
			cw = &noopCacheWarmer{}
			ctx = context.Background()
		})

		It("tests simple scan", func() {
			err := ds.MediaFolder(ctx).Put(&model.MediaFolder{
				ID:   path_hash.PathToMd5Hash("tests/fixtures"),
				Path: "tests/fixtures",
			})
			Expect(err).To(BeNil())

			progress := make(chan uint32, 100)

			scanner := NewTagScanner("tests/fixtures", ds, pls, cw)
			count, err := scanner.Scan(ctx, time.Time{}, progress)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(int64(11)))

			expected := model.MediaFolders{
				{
					ID:   "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
					Path: "tests/fixtures",
				},
				{
					ID:       "0ab3f79207dd4a2eeb529afea0932a14",
					Name:     "$Recycle.Bin",
					Path:     "tests/fixtures/$Recycle.Bin",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "d5aea55d20dca72cca2eaeb27caf4a2f",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "2296dc9dbe127641d2718d9b0290c5c8",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "a83ae8566962c51893e49739313711d7",
					Name:     "subfolder1",
					Path:     "tests/fixtures/playlists/subfolder1",
					ParentId: "502f80896b1adf58639fbe692c5f24bc",
				},
				{
					ID:       "859c18628cdac7e7fbb010133bc32729",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "ec4f4aa59145d8937934c7387645ff71",
					Name:     "artist",
					Path:     "tests/fixtures/artist",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "502f80896b1adf58639fbe692c5f24bc",
					Name:     "playlists",
					Path:     "tests/fixtures/playlists",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "51c285f7f7390da6aa5182e646a120b0",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "0129613054dcf67242ecc3fa8da90eb4",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "563817eba0c252d6f4fb50b187c340cf",
					Name:     "symlink2dir",
					Path:     "tests/fixtures/symlink2dir",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "faec25a6ba7a54edfeefdb71ab596623",
					Name:     "...unhidden_folder",
					Path:     "tests/fixtures/...unhidden_folder",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "671f1036b3fcd474946870cfe736a0a5",
					Name:     "an-album",
					Path:     "tests/fixtures/artist/an-album",
					ParentId: "ec4f4aa59145d8937934c7387645ff71",
				},
				{
					ID:       "ba7abd9ee1c34936ad6c30fb37250741",
					ParentId: "671f1036b3fcd474946870cfe736a0a5",
				},
				{
					ID:       "eb9101a0e5cde3b5477fd152e4ad1844",
					Name:     "empty_folder",
					Path:     "tests/fixtures/empty_folder",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "abe671b73251ea7d5877e48900f192c1",
					Name:     "subfolder2",
					Path:     "tests/fixtures/playlists/subfolder2",
					ParentId: "502f80896b1adf58639fbe692c5f24bc",
				},
				{
					ID:       "f7d67d0f9706769e59fae88a82065cf8",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "9101d69d497589e7b91938be55da4e1f",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "b1281a55b8e7dde7870c2a338178d9ef",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "fb951fa61daca15a9c76879d82e2c18e",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
				{
					ID:       "f25d647bb956bbca9e98ec313c08457a",
					ParentId: "da0b2b0b955ea5f4cd5a9eeaadb8cb79",
				},
			}
			folders := map[string]model.MediaFolder{}
			for _, folder := range expected {
				folders[folder.ID] = folder
			}

			dirs, err := ds.MediaFolder(ctx).GetAllDirectories()
			Expect(err).To(BeNil())
			Expect(len(dirs)).To(Equal(len(folders)))

			for _, folder := range dirs {
				mapped, ok := folders[folder.ID]
				if !ok {
					Expect(folder).To(BeNil())
				}
				Expect(ok).To(BeTrue())
				Expect(mapped).To(Equal(folder))
			}

			// Second scan
			count, err = scanner.Scan(ctx, time.Time{}, progress)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(int64(11)))

			dirs, err = ds.MediaFolder(ctx).GetAllDirectories()
			Expect(err).To(BeNil())
			Expect(len(dirs)).To(Equal(len(folders)))

			for _, folder := range dirs {
				mapped, ok := folders[folder.ID]
				Expect(ok).To(BeTrue())
				Expect(mapped).To(Equal(folder))
			}
		})
	})
})
