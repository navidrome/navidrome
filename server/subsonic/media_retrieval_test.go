package subsonic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRetrievalController", func() {
	var router *Router
	var ds model.DataStore
	mockRepo := &mockedMediaFile{}
	var artwork *fakeArtwork
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		ds = &tests.MockDataStore{
			MockedMediaFile: mockRepo,
		}
		artwork = &fakeArtwork{}
		router = New(ds, artwork, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w = httptest.NewRecorder()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.LyricsPriority = "embedded,.lrc"
	})

	Describe("GetCoverArt", func() {
		It("should return data for that id", func() {
			artwork.data = "image data"
			r := newGetRequest("id=34", "size=128")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(BeNil())
			Expect(artwork.recvId).To(Equal("34"))
			Expect(artwork.recvSize).To(Equal(128))
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should return placeholder if id parameter is missing (mimicking Subsonic)", func() {
			r := newGetRequest()
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(BeNil())
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should fail when the file is not found", func() {
			artwork.err = model.ErrNotFound
			r := newGetRequest("id=34", "size=128")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(MatchError("Artwork not found"))
		})

		It("should fail when there is an unknown error", func() {
			artwork.err = errors.New("weird error")
			r := newGetRequest("id=34", "size=128")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(MatchError("weird error"))
		})
	})

	Describe("GetLyrics", func() {
		It("should return data for given artist & title", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			lyrics, _ := model.ToLyrics("eng", "[00:18.80]We're no strangers to love\n[00:22.80]You know the rules and so do I")
			lyricsJson, err := json.Marshal(model.LyricList{
				*lyrics,
			})
			Expect(err).ToNot(HaveOccurred())

			mockRepo.SetData(model.MediaFiles{
				{
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
					Lyrics: string(lyricsJson),
				},
			})
			response, err := router.GetLyrics(r)
			if err != nil {
				log.Error("You're missing something.", err)
			}
			Expect(err).To(BeNil())
			Expect(response.Lyrics.Artist).To(Equal("Rick Astley"))
			Expect(response.Lyrics.Title).To(Equal("Never Gonna Give You Up"))
			Expect(response.Lyrics.Value).To(Equal("We're no strangers to love\nYou know the rules and so do I\n"))
		})
		It("should return empty subsonic response if the record corresponding to the given artist & title is not found", func() {
			r := newGetRequest("artist=Dheeraj", "title=Rinkiya+Ke+Papa")
			mockRepo.SetData(model.MediaFiles{})
			response, err := router.GetLyrics(r)
			if err != nil {
				log.Error("You're missing something.", err)
			}
			Expect(err).To(BeNil())
			Expect(response.Lyrics.Artist).To(Equal(""))
			Expect(response.Lyrics.Title).To(Equal(""))
			Expect(response.Lyrics.Value).To(Equal(""))
		})
		It("should return lyric file when finding mediafile with no embedded lyrics but present on filesystem", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			mockRepo.SetData(model.MediaFiles{
				{
					Path:   "tests/fixtures/test.mp3",
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
				},
			})
			response, err := router.GetLyrics(r)
			Expect(err).To(BeNil())
			Expect(response.Lyrics.Artist).To(Equal("Rick Astley"))
			Expect(response.Lyrics.Title).To(Equal("Never Gonna Give You Up"))
			Expect(response.Lyrics.Value).To(Equal("We're no strangers to love\nYou know the rules and so do I\n"))
		})
	})

	Describe("getLyricsBySongId", func() {
		const syncedLyrics = "[00:18.80]We're no strangers to love\n[00:22.801]You know the rules and so do I"
		const unsyncedLyrics = "We're no strangers to love\nYou know the rules and so do I"
		const metadata = "[ar:Rick Astley]\n[ti:That one song]\n[offset:-100]"
		var times = []int64{18800, 22801}

		compareResponses := func(actual *responses.LyricsList, expected responses.LyricsList) {
			Expect(actual).ToNot(BeNil())
			Expect(actual.StructuredLyrics).To(HaveLen(len(expected.StructuredLyrics)))
			for i, realLyric := range actual.StructuredLyrics {
				expectedLyric := expected.StructuredLyrics[i]

				Expect(realLyric.DisplayArtist).To(Equal(expectedLyric.DisplayArtist))
				Expect(realLyric.DisplayTitle).To(Equal(expectedLyric.DisplayTitle))
				Expect(realLyric.Lang).To(Equal(expectedLyric.Lang))
				Expect(realLyric.Synced).To(Equal(expectedLyric.Synced))

				if expectedLyric.Offset == nil {
					Expect(realLyric.Offset).To(BeNil())
				} else {
					Expect(*realLyric.Offset).To(Equal(*expectedLyric.Offset))
				}

				Expect(realLyric.Line).To(HaveLen(len(expectedLyric.Line)))
				for j, realLine := range realLyric.Line {
					expectedLine := expectedLyric.Line[j]
					Expect(realLine.Value).To(Equal(expectedLine.Value))

					if expectedLine.Start == nil {
						Expect(realLine.Start).To(BeNil())
					} else {
						Expect(*realLine.Start).To(Equal(*expectedLine.Start))
					}
				}
			}
		}

		It("should return mixed lyrics", func() {
			r := newGetRequest("id=1")
			synced, _ := model.ToLyrics("eng", syncedLyrics)
			unsynced, _ := model.ToLyrics("xxx", unsyncedLyrics)
			lyricsJson, err := json.Marshal(model.LyricList{
				*synced, *unsynced,
			})
			Expect(err).ToNot(HaveOccurred())

			mockRepo.SetData(model.MediaFiles{
				{
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
					Lyrics: string(lyricsJson),
				},
			})

			response, err := router.GetLyricsBySongId(r)
			Expect(err).ToNot(HaveOccurred())
			compareResponses(response.LyricsList, responses.LyricsList{
				StructuredLyrics: responses.StructuredLyrics{
					{
						Lang:          "eng",
						DisplayArtist: "Rick Astley",
						DisplayTitle:  "Never Gonna Give You Up",
						Synced:        true,
						Line: []responses.Line{
							{
								Start: &times[0],
								Value: "We're no strangers to love",
							},
							{
								Start: &times[1],
								Value: "You know the rules and so do I",
							},
						},
					},
					{
						Lang:          "xxx",
						DisplayArtist: "Rick Astley",
						DisplayTitle:  "Never Gonna Give You Up",
						Synced:        false,
						Line: []responses.Line{
							{
								Value: "We're no strangers to love",
							},
							{
								Value: "You know the rules and so do I",
							},
						},
					},
				},
			})
		})

		It("should parse lrc metadata", func() {
			r := newGetRequest("id=1")
			synced, _ := model.ToLyrics("eng", metadata+"\n"+syncedLyrics)
			lyricsJson, err := json.Marshal(model.LyricList{
				*synced,
			})
			Expect(err).ToNot(HaveOccurred())
			mockRepo.SetData(model.MediaFiles{
				{
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
					Lyrics: string(lyricsJson),
				},
			})

			response, err := router.GetLyricsBySongId(r)
			Expect(err).ToNot(HaveOccurred())

			offset := int64(-100)
			compareResponses(response.LyricsList, responses.LyricsList{
				StructuredLyrics: responses.StructuredLyrics{
					{
						DisplayArtist: "Rick Astley",
						DisplayTitle:  "That one song",
						Lang:          "eng",
						Synced:        true,
						Line: []responses.Line{
							{
								Start: &times[0],
								Value: "We're no strangers to love",
							},
							{
								Start: &times[1],
								Value: "You know the rules and so do I",
							},
						},
						Offset: &offset,
					},
				},
			})
		})
	})
})

type fakeArtwork struct {
	artwork.Artwork
	data     string
	err      error
	recvId   string
	recvSize int
}

func (c *fakeArtwork) GetOrPlaceholder(_ context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error) {
	if c.err != nil {
		return nil, time.Time{}, c.err
	}
	c.recvId = id
	c.recvSize = size
	return io.NopCloser(bytes.NewReader([]byte(c.data))), time.Time{}, nil
}

type mockedMediaFile struct {
	model.MediaFileRepository
	data model.MediaFiles
}

func (m *mockedMediaFile) SetData(mfs model.MediaFiles) {
	m.data = mfs
}

func (m *mockedMediaFile) GetAll(...model.QueryOptions) (model.MediaFiles, error) {
	return m.data, nil
}

func (m *mockedMediaFile) Get(id string) (*model.MediaFile, error) {
	for _, mf := range m.data {
		if mf.ID == id {
			return &mf, nil
		}
	}
	return nil, model.ErrNotFound
}
