package scanner_test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Scans a real library folder on the native filesystem (so Windows paths go through the OS path
// handling) and checks what reaches the DB in phase 1 (num_playlists) and phase 4 (playlists).
var _ = Describe("Scanner - PlaylistsPath", Ordered, ContinueOnFailure, func() {
	var ctx context.Context
	var ds model.DataStore
	var s model.Scanner
	var libPath string

	BeforeAll(func() {
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "123", IsAdmin: true})
		// The DB stays open until the suite ends, and Windows can't delete an open file
		tmpDir, err := os.MkdirTemp("", "scanner-playlists-path-test")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })
		conf.Server.DbPath = filepath.Join(tmpDir, "test-scanner.db?_journal_mode=WAL")
		db.Db().SetMaxOpenConns(1)
	})

	writeNSP := func(relPath, name string) {
		GinkgoHelper()
		full := filepath.Join(libPath, filepath.FromSlash(relPath))
		Expect(os.MkdirAll(filepath.Dir(full), 0755)).To(Succeed())
		nsp := `{"name": "` + name + `", "all": [{"is": {"loved": true}}]}`
		Expect(os.WriteFile(full, []byte(nsp), 0600)).To(Succeed())
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		libPath = GinkgoT().TempDir()
		conf.Server.MusicFolder = libPath
		conf.Server.DevExternalScanner = false
		conf.Server.AutoImportPlaylists = true

		db.Init(ctx)
		DeferCleanup(func() {
			Expect(tests.ClearDB()).To(Succeed())
		})
		ds = persistence.New(db.Db())

		adminUser := model.User{ID: "123", UserName: "admin", Name: "Admin User", IsAdmin: true, NewPassword: "password"}
		Expect(ds.User().Put(ctx, &adminUser)).To(Succeed())

		lib := model.Library{ID: 1, Name: "Native Library", Path: libPath}
		Expect(ds.Library().Put(ctx, &lib)).To(Succeed())

		s = scanner.New(ctx, ds, events.NoopBroker(),
			playlists.NewPlaylists(ds, artwork.NewUploader(ds)), metrics.NewNoopInstance())
	})

	scan := func(fullScan bool) {
		GinkgoHelper()
		_, err := s.ScanAll(ctx, fullScan)
		Expect(err).ToNot(HaveOccurred())
	}

	// One map, so a failure shows both the phase 1 and the phase 4 results
	results := func() map[string][]string {
		GinkgoHelper()
		folders, err := ds.Folder().GetAll(ctx)
		Expect(err).ToNot(HaveOccurred())
		var withPlaylists []string
		for _, f := range folders {
			if f.NumPlaylists > 0 {
				withPlaylists = append(withPlaylists, path.Join(f.Path, f.Name))
			}
		}
		all, err := ds.Playlist().GetAll(ctx)
		Expect(err).ToNot(HaveOccurred())
		var names []string
		for _, p := range all {
			Expect(p.Path).To(HavePrefix(libPath))
			names = append(names, p.Name)
		}
		return map[string][]string{
			"phase 1: folders with num_playlists > 0": slices.Sorted(slices.Values(withPlaylists)),
			"phase 4: imported playlists":             slices.Sorted(slices.Values(names)),
		}
	}

	expectResults := func(folders, names []string) {
		GinkgoHelper()
		Expect(results()).To(Equal(map[string][]string{
			"phase 1: folders with num_playlists > 0": slices.Sorted(slices.Values(folders)),
			"phase 4: imported playlists":             slices.Sorted(slices.Values(names)),
		}))
	}

	onWindows := func(values ...string) []string {
		if runtime.GOOS == "windows" {
			return values
		}
		return nil
	}

	// Phase 4 keeps its folder cursor open while importing, so with the suite's single DB connection
	// a scan stalls past ~6 playlist folders. Each library below stays under that.
	Describe("selecting nested folders", func() {
		BeforeEach(func() {
			writeNSP("Root.nsp", "Root")
			writeNSP("Playlists/navidrome/Rock.nsp", "Rock")
			writeNSP("Playlists/navidrome/Deep/Nested.nsp", "Nested")
			writeNSP("Playlists/other/Other.nsp", "Other")
		})

		DescribeTable("imports only playlists inside PlaylistsPath",
			func(pattern string, folders, names []string) {
				conf.Server.PlaylistsPath = pattern
				scan(true)
				expectResults(folders, names)
			},
			Entry("empty (default) imports everything", "",
				[]string{".", "Playlists/navidrome", "Playlists/navidrome/Deep", "Playlists/other"},
				[]string{"Root", "Rock", "Nested", "Other"}),
			Entry("nested folder", "Playlists/navidrome",
				[]string{"Playlists/navidrome"}, []string{"Rock"}),
			Entry("root and a nested ** pattern", "."+string(filepath.ListSeparator)+"Playlists/navidrome/**",
				[]string{".", "Playlists/navidrome", "Playlists/navidrome/Deep"}, []string{"Root", "Rock", "Nested"}),
			Entry("non-matching pattern imports nothing", "Music/**", nil, nil),
			// Backslash is a path separator on Windows (except in escaped brackets or braces), an escape elsewhere
			Entry("backslash nested folder (issue #6276 config)", `Playlists\navidrome`,
				onWindows("Playlists/navidrome"), onWindows("Rock")),
			Entry("backslash separator before braces", `Playlists\{navidrome,other}`,
				onWindows("Playlists/navidrome", "Playlists/other"), onWindows("Rock", "Other")),
		)
	})

	Describe("selecting folders with brackets in their names", func() {
		BeforeEach(func() {
			writeNSP("[Mix]/Mix.nsp", "Mix")
			writeNSP("Mix/Plain.nsp", "Plain")
			writeNSP("Playlists/[Mix]/NestedMix.nsp", "NestedMix")
			writeNSP("{Mix}/Braces.nsp", "Braces")
		})

		DescribeTable("imports only playlists inside PlaylistsPath",
			func(pattern string, folders, names []string) {
				conf.Server.PlaylistsPath = pattern
				scan(true)
				expectResults(folders, names)
			},
			Entry("brackets: empty (default) imports everything", "",
				[]string{"[Mix]", "Mix", "Playlists/[Mix]", "{Mix}"}, []string{"Mix", "Plain", "NestedMix", "Braces"}),
			Entry("brackets: escaped, top-level", `\[Mix\]`, []string{"[Mix]"}, []string{"Mix"}),
			Entry("brackets: escaped, nested", `Playlists/\[Mix\]`, []string{"Playlists/[Mix]"}, []string{"NestedMix"}),
			Entry("brackets: character class literal", `[[]Mix]`, []string{"[Mix]"}, []string{"Mix"}),
			Entry("brackets: unescaped brackets are a character class", `[Mix]`, nil, nil),
			Entry("brackets: backslash before escaped brackets, nested", `Playlists\\[Mix\]`,
				onWindows("Playlists/[Mix]"), onWindows("NestedMix")),
			Entry("brackets: backslash separator before a character class", `Playlists\[[]Mix]`,
				onWindows("Playlists/[Mix]"), onWindows("NestedMix")),
			Entry("brackets: escaped braces", `\{Mix\}`, []string{"{Mix}"}, []string{"Braces"}),
		)
	})

	// Folders indexed while their playlists were ignored keep num_playlists = 0. GC purges a folder
	// left with nothing else, but a cover image or a subfolder keeps its row.
	Describe("recovering folders indexed while their playlists were ignored", func() {
		indexIgnoringPlaylists := func() map[string]model.Folder {
			GinkgoHelper()
			conf.Server.PlaylistsPath = "Music/**"
			scan(true)
			expectResults(nil, nil)
			folders, err := ds.Folder().GetAll(ctx)
			Expect(err).ToNot(HaveOccurred())
			stored := map[string]model.Folder{}
			for _, f := range folders {
				Expect(f.NumPlaylists).To(BeZero())
				stored[path.Join(f.Path, f.Name)] = f
			}
			return stored
		}

		It("recovery: playlist-only folder, unchanged quick scan", func() {
			writeNSP("Playlists/navidrome/Rock.nsp", "Rock")
			Expect(indexIgnoringPlaylists()).ToNot(HaveKey("Playlists/navidrome"), "GC purged the folder")

			conf.Server.PlaylistsPath = "Playlists/navidrome"
			scan(false)
			expectResults([]string{"Playlists/navidrome"}, []string{"Rock"})
		})

		Describe("folder kept by a playlist cover image", func() {
			var stored map[string]model.Folder

			BeforeEach(func() {
				writeNSP("Playlists/navidrome/Rock.nsp", "Rock")
				Expect(os.WriteFile(filepath.Join(libPath, "Playlists", "navidrome", "Rock.jpg"), []byte("synthetic"), 0600)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(libPath, "Art"), 0755)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(libPath, "Art", "cover.jpg"), []byte("synthetic"), 0600)).To(Succeed())
				stored = indexIgnoringPlaylists()
				Expect(stored).To(HaveKey("Playlists/navidrome"))
				Expect(stored["Playlists/navidrome"].Path).To(Equal("Playlists"))
				Expect(stored["Playlists/navidrome"].ImageFiles).To(ConsistOf("Rock.jpg"))
				Expect(stored).To(HaveKey("Art"))
			})

			It("recovery: cover image folder, unchanged quick scan", func() {
				conf.Server.PlaylistsPath = "Playlists/navidrome"
				scan(false)
				expectResults([]string{"Playlists/navidrome"}, []string{"Rock"})
			})

			It("recovery: cover image folder, full scan", func() {
				conf.Server.PlaylistsPath = "Playlists/navidrome"
				scan(true)
				expectResults([]string{"Playlists/navidrome"}, []string{"Rock"})
			})

			It("recovery: cover image folder, quick scan after touching the playlist", func() {
				conf.Server.PlaylistsPath = "Playlists/navidrome"
				later := time.Now().Add(time.Minute)
				Expect(os.Chtimes(filepath.Join(libPath, "Playlists", "navidrome", "Rock.nsp"), later, later)).To(Succeed())
				scan(false)
				expectResults([]string{"Playlists/navidrome"}, []string{"Rock"})
			})

			It("recovery: still-excluded folder stays ignored on a quick scan", func() {
				conf.Server.PlaylistsPath = "Other/**"
				scan(false)
				expectResults(nil, nil)
			})

			It("recovery: folder without playlist files is not rescanned", func() {
				conf.Server.PlaylistsPath = "Playlists/navidrome"
				scan(false)
				art, err := ds.Folder().Get(ctx, stored["Art"].ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(art.Hash).To(Equal(stored["Art"].Hash))
				Expect(art.UpdateAt).To(BeTemporally("==", stored["Art"].UpdateAt))
			})

			It("recovery: quick scan stops counting playlists PlaylistsPath no longer includes", func() {
				conf.Server.PlaylistsPath = "Playlists/navidrome"
				scan(false)
				expectResults([]string{"Playlists/navidrome"}, []string{"Rock"})

				conf.Server.PlaylistsPath = "Other/**"
				scan(false)
				// Already imported playlists stay; only the folder count changes
				expectResults(nil, []string{"Rock"})
			})
		})

		It("recovery: parent folder with a subfolder, unchanged quick scan", func() {
			writeNSP("Playlists/navidrome/Rock.nsp", "Rock")
			writeNSP("Playlists/navidrome/Deep/Nested.nsp", "Nested")
			stored := indexIgnoringPlaylists()
			Expect(stored).To(HaveKey("Playlists/navidrome"), "kept as the parent of Deep")
			Expect(stored).ToNot(HaveKey("Playlists/navidrome/Deep"), "GC purged the leaf")

			conf.Server.PlaylistsPath = "Playlists/navidrome/**"
			scan(false)
			expectResults([]string{"Playlists/navidrome", "Playlists/navidrome/Deep"}, []string{"Rock", "Nested"})
		})
	})
})
