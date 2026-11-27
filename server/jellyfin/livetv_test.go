package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Live TV radio", func() {
	var ctx context.Context
	var ds model.DataStore
	var api *Router
	var pt *fakePlayTracker
	var art *fakeArtwork
	var token string

	putRadio := func(rd model.Radio) {
		GinkgoHelper()
		adminCtx := request.WithUser(ctx, model.User{IsAdmin: true})
		Expect(ds.Radio().Put(adminCtx, &rd)).To(Succeed())
	}

	call := func(method, path, body string) *httptest.ResponseRecorder {
		GinkgoHelper()
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
		r.Header.Set("X-Emby-Authorization", `MediaBrowser Client="WatchAmp", Device="Watch", DeviceId="dev-1", Version="1.0"`)
		r.Header.Set("X-Emby-Token", token)
		api.ServeHTTP(w, r)
		return w
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		auth.Init(ds)
		persistencetest.SetLibraries(ctx, ds, "/music1")
		usr := model.User{ID: testID("u1"), UserName: "alice", NewPassword: "secret"}
		Expect(ds.User().Put(ctx, &usr)).To(Succeed())
		Expect(ds.User().SetUserLibraries(ctx, usr.ID, []int{1})).To(Succeed())
		var err error
		token, err = auth.CreateToken(&usr)
		Expect(err).ToNot(HaveOccurred())

		pt = &fakePlayTracker{}
		art = &fakeArtwork{}
		api = New(ds, art, nil, nil, &fakePlayers{}, pt, nil, nil, nil, nil, nil, nil)

		putRadio(model.Radio{ID: testID("r-zeta"), Name: "Zeta FM", StreamUrl: "https://zeta.example.org/live.aac"})
		putRadio(model.Radio{ID: testID("r-alpha"), Name: "Alpha Radio", StreamUrl: "https://alpha.example.org/stream.mp3", UploadedImage: "alpha.png"})
	})

	stationID := func(label string) string { return dto.EncodeID(testID(label)) }

	Describe("GET /LiveTv/Channels", func() {
		list := func(query string) dto.QueryResult {
			GinkgoHelper()
			w := call(http.MethodGet, "/LiveTv/Channels"+query, "")
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			return res
		}

		It("lists the stations as radio channels sorted by name", func() {
			res := list("?type=Radio&userId=" + testID("u1"))

			Expect(res.TotalRecordCount).To(Equal(2))
			Expect(res.Items).To(HaveLen(2))
			first := res.Items[0]
			Expect(first.Name).To(Equal("Alpha Radio"))
			Expect(first.Id).To(Equal(stationID("r-alpha")))
			Expect(first.Type).To(Equal("TvChannel"))
			Expect(first.ChannelType).To(Equal("Radio"))
			Expect(first.MediaType).To(Equal("Audio"))
			Expect(first.IsFolder).To(BeFalse())
			Expect(res.Items[1].Name).To(Equal("Zeta FM"))
		})

		It("advertises a Primary image only for stations with an uploaded image", func() {
			res := list("")

			Expect(res.Items[0].ImageTags).To(HaveKey("Primary"))
			Expect(res.Items[1].ImageTags).ToNot(HaveKey("Primary"))
		})

		It("pages with StartIndex and Limit", func() {
			res := list("?StartIndex=1&Limit=1")

			Expect(res.TotalRecordCount).To(Equal(2))
			Expect(res.StartIndex).To(Equal(1))
			Expect(res.Items).To(HaveLen(1))
			Expect(res.Items[0].Name).To(Equal("Zeta FM"))
		})

		It("returns no stations when the client asks only for TV channels", func() {
			res := list("?type=TV")

			Expect(res.TotalRecordCount).To(BeZero())
			Expect(res.Items).To(BeEmpty())
		})
	})

	It("resolves a station as a single item", func() {
		for _, path := range []string{"/Items/" + stationID("r-zeta"), "/Users/" + testID("u1") + "/Items/" + stationID("r-zeta")} {
			w := call(http.MethodGet, path, "")
			Expect(w.Code).To(Equal(http.StatusOK), path)
			var item dto.BaseItemDto
			Expect(json.Unmarshal(w.Body.Bytes(), &item)).To(Succeed())
			Expect(item.Name).To(Equal("Zeta FM"), path)
			Expect(item.ChannelType).To(Equal("Radio"), path)
		}
	})

	It("reports a failed station lookup as a server error, not a missing item", func() {
		api = New(failingRadioDS{DataStore: ds}, art, nil, nil, &fakePlayers{}, pt, nil, nil, nil, nil, nil, nil)

		w := call(http.MethodPost, "/Items/"+stationID("r-zeta")+"/PlaybackInfo", `{}`)

		Expect(w.Code).To(Equal(http.StatusInternalServerError))
	})

	DescribeTable("PlaybackInfo returns the station URL for direct playback",
		func(method string) {
			w := call(method, "/Items/"+stationID("r-zeta")+"/PlaybackInfo", `{"DeviceProfile":{}}`)

			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.PlaybackInfoResponse
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.MediaSources).To(HaveLen(1))
			src := res.MediaSources[0]
			Expect(src.Id).To(Equal(stationID("r-zeta")))
			Expect(src.Path).To(Equal("https://zeta.example.org/live.aac"))
			Expect(src.IsInfiniteStream).To(BeTrue())
			Expect(src.SupportsDirectPlay).To(BeTrue())
			Expect(src.SupportsTranscoding).To(BeFalse())
			Expect(src.RequiresOpening).To(BeFalse())
			Expect(src.TranscodingUrl).To(BeEmpty())
		},
		Entry("POST", http.MethodPost),
		Entry("GET", http.MethodGet),
	)

	// A redirect would carry the client's X-Emby-Token header to the station's host.
	DescribeTable("never sends a client to the station from a stream URL",
		func(method, path string) {
			w := call(method, strings.ReplaceAll(path, "{id}", stationID("r-alpha")), "")

			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(w.Header().Get("Location")).To(BeEmpty())
		},
		Entry("audio stream", http.MethodGet, "/Audio/{id}/stream"),
		Entry("audio stream with container", http.MethodGet, "/Audio/{id}/stream.mp3"),
		Entry("audio universal", http.MethodGet, "/Audio/{id}/universal"),
		Entry("audio stream HEAD", http.MethodHead, "/Audio/{id}/stream"),
		Entry("video stream", http.MethodGet, "/Videos/{id}/stream"),
	)

	It("serves a station's uploaded image", func() {
		w := call(http.MethodGet, "/Items/"+stationID("r-alpha")+"/Images/Primary", "")

		Expect(w.Code).To(Equal(http.StatusOK))
		rd, err := ds.Radio().Get(ctx, testID("r-alpha"))
		Expect(err).ToNot(HaveOccurred())
		Expect(art.recvId).To(Equal(rd.CoverArtID().String()))
	})

	DescribeTable("keeps station playback out of the scrobbler",
		func(path string) {
			w := call(http.MethodPost, path, `{"ItemId":"`+stationID("r-zeta")+`","PositionTicks":100000000}`)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(pt.reported).To(BeEmpty())
		},
		Entry("start", "/Sessions/Playing"),
		Entry("progress", "/Sessions/Playing/Progress"),
		Entry("stopped", "/Sessions/Playing/Stopped"),
	)

	It("still reports song playback to the scrobbler", func() {
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: testID("s1"), Title: "Song", Suffix: "mp3", LibraryID: 1})

		w := call(http.MethodPost, "/Sessions/Playing", `{"ItemId":"`+dto.EncodeID(testID("s1"))+`"}`)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(pt.reported).To(HaveLen(1))
		Expect(pt.reported[0].MediaId).To(Equal(testID("s1")))
	})

	It("grants Live TV access in the user policy, so clients show the radio channels", func() {
		w := call(http.MethodGet, "/Users/Me", "")

		Expect(w.Code).To(Equal(http.StatusOK))
		var u dto.UserDto
		Expect(json.Unmarshal(w.Body.Bytes(), &u)).To(Succeed())
		Expect(u.Policy.EnableLiveTvAccess).To(BeTrue())
	})
})

type failingRadioDS struct{ model.DataStore }

func (failingRadioDS) Radio() model.RadioRepository { return failingRadioRepo{} }

type failingRadioRepo struct{ model.RadioRepository }

func (failingRadioRepo) Get(context.Context, string) (*model.Radio, error) {
	return nil, errors.New("database is locked")
}
