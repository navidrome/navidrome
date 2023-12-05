package persistence

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/beego/beego/v2/client/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	NewId string = "123-456-789"
)

var _ = Describe("RadioRepository", func() {
	var httpClient *tests.FakeHttpClient
	var repo model.RadioRepository

	testRadio := func(idx int, message string) {
		It(message, func() {
			radio := testRadios[idx]

			res, err := repo.Get(radio.ID)

			Expect(err).To(BeNil())
			Expect(res.ID).To(Equal(radio.ID))
			Expect(res.Links).To(HaveLen(len(radio.Links)))

			for i, link := range res.Links {
				Expect(link.Name).To(Equal(radio.Links[i].Name))
				Expect(link.Url).To(Equal(radio.Links[i].Url))
			}
		})
	}

	Describe("Admin User", func() {
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString("")),
				StatusCode: 200,
			}

			ctx := log.NewContext(context.TODO())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			repo = NewRadioRepository(ctx, orm.NewOrm())
			repo.(*radioRepository).client = httpClient
		})

		AfterEach(func() {
			all, _ := repo.GetAll()

			for _, radio := range all {
				_ = repo.Delete(radio.ID)
			}

			SetupRadio(repo.(*radioRepository), httpClient)
		})

		Describe("Count", func() {
			It("returns the number of radios in the DB", func() {
				Expect(repo.CountAll()).To(Equal(int64(5)))
			})
		})

		Describe("Delete", func() {
			It("deletes existing item", func() {
				err := repo.Delete(radioWithHomePage.ID)

				Expect(err).To(BeNil())

				_, err = repo.Get(radioWithHomePage.ID)
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("Get", func() {
			It("returns an existing playlist (no links)", func() {
				res, err := repo.Get(radioWithHomePage.ID)

				Expect(err).To(BeNil())
				Expect(res.ID).To(Equal(radioWithHomePage.ID))
				Expect(res.Links).To(HaveLen(0))
			})

			testRadio(2, "returns links from m3u playlist")
			testRadio(3, "returns links from m3u extended playlist")
			testRadio(4, "returns links from pls playlist")

			It("errors when missing", func() {
				_, err := repo.Get("notanid")

				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("GetAll", func() {
			It("returns all items from the DB", func() {
				all, err := repo.GetAll()
				Expect(err).To(BeNil())

				for i, radio := range testRadios {
					Expect(all[i].ID).To(Equal(radio.ID))
					Expect(all[i].StreamUrl).To(Equal(radio.StreamUrl))
					Expect(all[i].IsPlaylist).To(Equal(radio.IsPlaylist))
					Expect(all[i].Name).To(Equal(radio.Name))
				}
			})
		})

		Describe("Put", func() {
			Describe("Regular stream", func() {
				It("successfully updates item", func() {
					httpClient.Res = http.Response{
						Body:       io.NopCloser(bytes.NewBufferString("")),
						StatusCode: 200,
					}

					err := repo.Put(&model.Radio{
						ID:        radioWithHomePage.ID,
						Name:      "New Name",
						StreamUrl: "https://example.com:4533/app",
					})

					Expect(err).To(BeNil())

					item, err := repo.Get(radioWithHomePage.ID)
					Expect(err).To(BeNil())

					Expect(item.HomePageUrl).To(Equal(""))
				})

				It("successfully creates item", func() {
					httpClient.Res = http.Response{
						Body:       io.NopCloser(bytes.NewBufferString("")),
						StatusCode: 200,
					}

					err := repo.Put(&model.Radio{
						Name:      "New radio",
						StreamUrl: "https://example.com:4533/app",
					})

					Expect(err).To(BeNil())
					Expect(repo.CountAll()).To(Equal(int64(6)))

					all, err := repo.GetAll()
					Expect(err).To(BeNil())
					Expect(all[5].StreamUrl).To(Equal("https://example.com:4533/app"))
				})
			})
		})

		Describe("Playlist stream", func() {
			testCreate := func(filename string, contentType string, links *model.RadioLinks) {
				ext := path.Ext(filename)

				It("Successfully creates an "+ext+" stream", func() {
					f, _ := os.Open("tests/fixtures/radios/" + filename)
					header := http.Header{}
					header.Set("Content-Type", contentType)
					httpClient.Res = http.Response{
						Body: f, StatusCode: 200, Header: header,
					}

					name := ext + " stream"
					url := "https://example.com:1234/" + filename

					err := repo.Put(&model.Radio{
						Name:      name,
						StreamUrl: url,
					})

					Expect(err).To(BeNil())
					Expect(repo.CountAll()).To(Equal(int64(6)))

					all, err := repo.GetAll()
					Expect(err).To(BeNil())
					Expect(all[5].Name).To(Equal(name))
					Expect(all[5].StreamUrl).To(Equal(url))
					Expect(all[5].IsPlaylist).To(BeTrue())

					radio, err := repo.Get(all[5].ID)
					Expect(err).To(BeNil())

					Expect(radio.Links).To(HaveLen(len(*links)))

					for i, link := range *links {
						Expect(radio.Links[i].Name).To(Equal(link.Name))
						Expect(radio.Links[i].Url).To(Equal(link.Url))
					}
				})
			}

			testCreate("radio.m3u", "audio/mpegurl", &m3uLinks)
			testCreate("radio-extended.m3u", "audio/x-mpegurl", &m3uExtendedLinks)
			testCreate("radio.pls", "audio/x-scpls", &plsLinks)

			testUpdate := func(id string, filename string, contentType string, links *model.RadioLinks) {
				ext := path.Ext(filename)

				It("Successfully updates an "+ext+" stream", func() {
					f, _ := os.Open("tests/fixtures/radios/" + filename)
					header := http.Header{}
					header.Set("Content-Type", contentType)
					httpClient.Res = http.Response{
						Body: f, StatusCode: 200, Header: header,
					}

					name := ext + " stream"
					url := "https://example.com:1234/" + filename

					err := repo.Put(&model.Radio{
						ID:        id,
						Name:      name,
						StreamUrl: url,
					})

					Expect(err).To(BeNil())
					Expect(repo.CountAll()).To(Equal(int64(5)))

					radio, err := repo.Get(id)
					Expect(err).To(BeNil())
					Expect(radio.IsPlaylist).To(BeTrue())

					Expect(radio.Name).To(Equal(name))
					Expect(radio.StreamUrl).To(Equal(url))
					Expect(radio.IsPlaylist).To(BeTrue())
					Expect(radio.Links).To(HaveLen(len(*links)))

					for i, link := range *links {
						Expect(radio.Links[i].Name).To(Equal(link.Name))
						Expect(radio.Links[i].Url).To(Equal(link.Url))
					}
				})
			}

			testUpdate("3456", "radio.m3u", "application/vnd.apple.mpegurl", &m3uLinks)
			testUpdate("1234", "radio-extended.m3u", "application/vnd.apple.mpegurl.audio", &m3uExtendedLinks)
			testUpdate("2345", "radio.pls", "audio/x-scpls", &plsLinks)
		})
	})

	Describe("Regular User", func() {
		BeforeEach(func() {
			ctx := log.NewContext(context.TODO())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: false})
			repo = NewRadioRepository(ctx, orm.NewOrm())
		})

		Describe("Count", func() {
			It("returns the number of radios in the DB", func() {
				Expect(repo.CountAll()).To(Equal(int64(5)))
			})
		})

		Describe("Delete", func() {
			It("fails to delete items", func() {
				err := repo.Delete(radioWithHomePage.ID)

				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("Get", func() {
			It("returns an existing playlist (no links)", func() {
				res, err := repo.Get(radioWithHomePage.ID)

				Expect(err).To(BeNil())
				Expect(res.ID).To(Equal(radioWithHomePage.ID))
				Expect(res.Links).To(HaveLen(0))
			})

			testRadio(2, "returns links from m3u playlist")
			testRadio(3, "returns links from m3u extended playlist")
			testRadio(4, "returns links from pls playlist")

			It("errors when missing", func() {
				_, err := repo.Get("notanid")

				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("GetAll", func() {
			It("returns all items from the DB", func() {
				all, err := repo.GetAll()
				Expect(err).To(BeNil())

				for i, radio := range testRadios {
					Expect(all[i].ID).To(Equal(radio.ID))
					Expect(all[i].StreamUrl).To(Equal(radio.StreamUrl))
					Expect(all[i].IsPlaylist).To(Equal(radio.IsPlaylist))
					Expect(all[i].Name).To(Equal(radio.Name))
				}
			})
		})

		Describe("Put", func() {
			It("fails to update item", func() {
				err := repo.Put(&model.Radio{
					ID:        radioWithHomePage.ID,
					Name:      "New Name",
					StreamUrl: "https://example.com:4533/app",
				})

				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})
	})
})
