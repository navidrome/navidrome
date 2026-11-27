package cmd

import (
	"errors"
	"strings"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("writeMissingList", func() {
	cursor := func(err error, mfs ...model.MediaFile) model.MediaFileCursor {
		return func(yield func(model.MediaFile, error) bool) {
			for _, mf := range mfs {
				if !yield(mf, nil) {
					return
				}
			}
			if err != nil {
				yield(model.MediaFile{}, err)
			}
		}
	}
	song := model.MediaFile{ID: "1", LibraryID: 1, Path: "Bach: Goldberg/01.mp3", Title: "Aria", Album: "Goldberg", Artist: "Bach"}

	It("writes csv with a header, quoting as needed", func() {
		var out strings.Builder
		Expect(writeMissingList(&out, "csv", cursor(nil, song))).To(Succeed())
		Expect(out.String()).To(Equal("id,library id,title,album,artist,path\n1,1,Aria,Goldberg,Bach,Bach: Goldberg/01.mp3\n"))
	})

	It("writes a json array", func() {
		var out strings.Builder
		Expect(writeMissingList(&out, "json", cursor(nil, song, song))).To(Succeed())
		Expect(out.String()).To(MatchJSON(`[
			{"id":"1","libraryId":1,"path":"Bach: Goldberg/01.mp3","title":"Aria","album":"Goldberg","artist":"Bach"},
			{"id":"1","libraryId":1,"path":"Bach: Goldberg/01.mp3","title":"Aria","album":"Goldberg","artist":"Bach"}
		]`))
	})

	It("writes an empty json array when nothing is missing", func() {
		var out strings.Builder
		Expect(writeMissingList(&out, "json", cursor(nil))).To(Succeed())
		Expect(out.String()).To(MatchJSON(`[]`))
	})

	It("returns the cursor's error", func() {
		var out strings.Builder
		Expect(writeMissingList(&out, "csv", cursor(errors.New("boom"), song))).To(MatchError("boom"))
	})
})

var _ = Describe("preferQualified", func() {
	target := model.MediaFile{ID: "want", LibraryID: 1, Path: "foo.mp3"}
	decoy := model.MediaFile{ID: "decoy", LibraryID: 1, Path: "1:foo.mp3"}

	It("picks the library-qualified match over a literal path that looks like one", func() {
		Expect(preferQualified("1:foo.mp3", model.MediaFiles{target, decoy})).To(Equal(model.MediaFiles{target}))
	})

	It("picks the named library when the same path exists in two", func() {
		other := model.MediaFile{ID: "other", LibraryID: 2, Path: "foo.mp3"}
		Expect(preferQualified("1:foo.mp3", model.MediaFiles{target, other})).To(Equal(model.MediaFiles{target}))
	})

	It("leaves an unqualified reference ambiguous", func() {
		both := model.MediaFiles{target, {ID: "other", LibraryID: 2, Path: "foo.mp3"}}
		Expect(preferQualified("foo.mp3", both)).To(Equal(both))
	})

	It("leaves it alone when the prefix is not a library id", func() {
		both := model.MediaFiles{decoy, {ID: "other", LibraryID: 2, Path: "1:foo.mp3"}}
		Expect(preferQualified("x:foo.mp3", both)).To(Equal(both))
	})

	It("leaves it alone when no candidate matches the qualified form", func() {
		both := model.MediaFiles{decoy, {ID: "other", LibraryID: 2, Path: "1:foo.mp3"}}
		Expect(preferQualified("9:nope.mp3", both)).To(Equal(both))
	})
})
