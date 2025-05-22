package nativeapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlists", func() {
	Describe("createPlaylistFromM3U", func() {
		It("returns BadRequest when import fails", func() {
			p := stubPlaylists{err: errors.New("invalid")}
			r := httptest.NewRequest(http.MethodPost, "/playlist", strings.NewReader("bad"))
			w := httptest.NewRecorder()

			createPlaylistFromM3U(p)(w, r)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns Created and playlist contents when import succeeds", func() {
			pls := &model.Playlist{Name: "pl"}
			p := stubPlaylists{result: pls}
			r := httptest.NewRequest(http.MethodPost, "/playlist", strings.NewReader("good"))
			w := httptest.NewRecorder()

			createPlaylistFromM3U(p)(w, r)

			Expect(w.Code).To(Equal(http.StatusCreated))
			Expect(w.Body.String()).To(Equal(pls.ToM3U8()))
		})
	})
})

type stubPlaylists struct {
	result *model.Playlist
	err    error
}

func (s stubPlaylists) ImportFile(ctx context.Context, folder *model.Folder, filename string) (*model.Playlist, error) {
	return nil, nil
}

func (s stubPlaylists) Update(ctx context.Context, playlistID string, name *string, comment *string, public *bool, idsToAdd []string, idxToRemove []int) error {
	return nil
}

func (s stubPlaylists) ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error) {
	return s.result, s.err
}
