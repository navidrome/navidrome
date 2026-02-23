package playlists

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseNSP", func() {
	var s *playlists
	ctx := context.Background()

	BeforeEach(func() {
		s = &playlists{}
	})

	It("parses a well-formed NSP with all fields", func() {
		nsp := `{
			"name": "My Smart Playlist",
			"comment": "A test playlist",
			"public": true,
			"all": [{"is": {"loved": true}}],
			"sort": "title",
			"order": "asc",
			"limit": 50
		}`
		pls := &model.Playlist{Name: "default-name"}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Name).To(Equal("My Smart Playlist"))
		Expect(pls.Comment).To(Equal("A test playlist"))
		Expect(pls.Public).To(BeTrue())
		Expect(pls.Rules).ToNot(BeNil())
		Expect(pls.Rules.Sort).To(Equal("title"))
		Expect(pls.Rules.Order).To(Equal("asc"))
		Expect(pls.Rules.Limit).To(Equal(50))
		Expect(pls.Rules.Expression).To(BeAssignableToTypeOf(criteria.All{}))
	})

	It("keeps existing name when NSP has no name field", func() {
		nsp := `{"all": [{"is": {"loved": true}}]}`
		pls := &model.Playlist{Name: "Original Name"}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Name).To(Equal("Original Name"))
	})

	It("keeps existing comment when NSP has no comment field", func() {
		nsp := `{"all": [{"is": {"loved": true}}]}`
		pls := &model.Playlist{Comment: "Original Comment"}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Comment).To(Equal("Original Comment"))
	})

	It("strips JSON comments before parsing", func() {
		nsp := `{
			// Line comment
			"name": "Commented Playlist",
			/* Block comment */
			"all": [{"is": {"loved": true}}]
		}`
		pls := &model.Playlist{}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Name).To(Equal("Commented Playlist"))
	})

	It("uses server default when public field is absent", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DefaultPlaylistPublicVisibility = true

		nsp := `{"all": [{"is": {"loved": true}}]}`
		pls := &model.Playlist{}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Public).To(BeTrue())
	})

	It("honors explicit public: false over server default", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DefaultPlaylistPublicVisibility = true

		nsp := `{"public": false, "all": [{"is": {"loved": true}}]}`
		pls := &model.Playlist{}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Public).To(BeFalse())
	})

	It("returns a syntax error with line and column info", func() {
		nsp := "{\n  \"name\": \"Bad\",\n  \"all\": [INVALID]\n}"
		pls := &model.Playlist{}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("JSON syntax error in SmartPlaylist"))
		Expect(err.Error()).To(MatchRegexp(`line \d+, column \d+`))
	})

	It("returns a parsing error for completely invalid JSON", func() {
		nsp := `not json at all`
		pls := &model.Playlist{}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("SmartPlaylist"))
	})

	It("gracefully handles non-string name field", func() {
		nsp := `{"name": 123, "all": [{"is": {"loved": true}}]}`
		pls := &model.Playlist{Name: "Original"}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).ToNot(HaveOccurred())
		// Type assertion in UnmarshalJSON fails silently; name stays as original
		Expect(pls.Name).To(Equal("Original"))
	})

	It("parses criteria with multiple rules", func() {
		nsp := `{
			"all": [
				{"is": {"loved": true}},
				{"contains": {"title": "rock"}}
			],
			"sort": "lastPlayed",
			"order": "desc",
			"limit": 100
		}`
		pls := &model.Playlist{}
		err := s.parseNSP(ctx, pls, strings.NewReader(nsp))
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Rules).ToNot(BeNil())
		Expect(pls.Rules.Sort).To(Equal("lastPlayed"))
		Expect(pls.Rules.Order).To(Equal("desc"))
		Expect(pls.Rules.Limit).To(Equal(100))
	})
})

var _ = Describe("getPositionFromOffset", func() {
	It("returns correct position on first line", func() {
		data := []byte("hello world")
		line, col := getPositionFromOffset(data, 5)
		Expect(line).To(Equal(1))
		Expect(col).To(Equal(5))
	})

	It("returns correct position after newlines", func() {
		data := []byte("line1\nline2\nline3")
		// Offsets: l(0) i(1) n(2) e(3) 1(4) \n(5) l(6) i(7) n(8)
		line, col := getPositionFromOffset(data, 8)
		Expect(line).To(Equal(2))
		Expect(col).To(Equal(3))
	})

	It("returns correct position at start of new line", func() {
		data := []byte("line1\nline2")
		// After \n at offset 5, col resets to 1; offset 6 is 'l' -> col=1
		line, col := getPositionFromOffset(data, 6)
		Expect(line).To(Equal(2))
		Expect(col).To(Equal(1))
	})

	It("handles multiple newlines", func() {
		data := []byte("a\nb\nc\nd")
		// a(0) \n(1) b(2) \n(3) c(4) \n(5) d(6)
		line, col := getPositionFromOffset(data, 6)
		Expect(line).To(Equal(4))
		Expect(col).To(Equal(1))
	})
})

var _ = Describe("newSyncedPlaylist", func() {
	var s *playlists

	BeforeEach(func() {
		s = &playlists{}
	})

	It("creates a synced playlist with correct attributes", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "test.m3u"), []byte("content"), 0600)).To(Succeed())

		pls, err := s.newSyncedPlaylist(tmpDir, "test.m3u")
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Name).To(Equal("test"))
		Expect(pls.Comment).To(Equal("Auto-imported from 'test.m3u'"))
		Expect(pls.Public).To(BeFalse())
		Expect(pls.Path).To(Equal(filepath.Join(tmpDir, "test.m3u")))
		Expect(pls.Sync).To(BeTrue())
		Expect(pls.UpdatedAt).ToNot(BeZero())
	})

	It("strips extension from filename to derive name", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "My Favorites.nsp"), []byte("{}"), 0600)).To(Succeed())

		pls, err := s.newSyncedPlaylist(tmpDir, "My Favorites.nsp")
		Expect(err).ToNot(HaveOccurred())
		Expect(pls.Name).To(Equal("My Favorites"))
	})

	It("returns error for non-existent file", func() {
		tmpDir := GinkgoT().TempDir()
		_, err := s.newSyncedPlaylist(tmpDir, "nonexistent.m3u")
		Expect(err).To(HaveOccurred())
	})
})
