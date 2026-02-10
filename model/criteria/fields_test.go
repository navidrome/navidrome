package criteria

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("fields", func() {
	Describe("mapFields", func() {
		It("ignores random fields", func() {
			m := map[string]any{"random": "123"}
			m = mapFields(m)
			gomega.Expect(m).To(gomega.BeEmpty())
		})
	})

	Describe("requiredJoins", func() {
		It("returns 0 for nil expression", func() {
			gomega.Expect(requiredJoins(nil)).To(gomega.Equal(JoinType(0)))
		})

		It("returns 0 for regular fields", func() {
			expr := All{Is{"title": "test"}, Gt{"playCount": 5}}
			gomega.Expect(requiredJoins(expr)).To(gomega.Equal(JoinType(0)))
		})

		It("returns JoinAlbumAnnotation for album annotation fields", func() {
			expr := All{Gt{"albumRating": 3}}
			gomega.Expect(requiredJoins(expr)).To(gomega.Equal(JoinAlbumAnnotation))
		})

		It("returns JoinArtistAnnotation for artist annotation fields", func() {
			expr := All{Gt{"artistRating": 3}}
			gomega.Expect(requiredJoins(expr)).To(gomega.Equal(JoinArtistAnnotation))
		})

		It("returns both join types when both are used", func() {
			expr := All{Gt{"albumRating": 3}, Is{"artistLoved": true}}
			gomega.Expect(requiredJoins(expr)).To(gomega.Equal(JoinAlbumAnnotation | JoinArtistAnnotation))
		})

		It("detects joins in nested Any expressions", func() {
			expr := All{
				Any{
					Gt{"albumRating": 3},
					Is{"title": "test"},
				},
			}
			gomega.Expect(requiredJoins(expr)).To(gomega.Equal(JoinAlbumAnnotation))
		})

		It("detects joins in deeply nested expressions", func() {
			expr := Any{
				All{
					Any{
						Is{"artistPlayCount": 10},
					},
				},
			}
			gomega.Expect(requiredJoins(expr)).To(gomega.Equal(JoinArtistAnnotation))
		})
	})

	Describe("requiredSortJoins", func() {
		It("returns 0 for regular sort fields", func() {
			gomega.Expect(requiredSortJoins("title")).To(gomega.Equal(JoinType(0)))
		})

		It("returns JoinAlbumAnnotation for album annotation sort", func() {
			gomega.Expect(requiredSortJoins("albumRating")).To(gomega.Equal(JoinAlbumAnnotation))
		})

		It("returns JoinArtistAnnotation for artist annotation sort", func() {
			gomega.Expect(requiredSortJoins("-artistRating")).To(gomega.Equal(JoinArtistAnnotation))
		})

		It("returns both join types for mixed sort fields", func() {
			gomega.Expect(requiredSortJoins("albumRating,-artistPlayCount")).To(
				gomega.Equal(JoinAlbumAnnotation | JoinArtistAnnotation))
		})

		It("returns 0 for empty sort", func() {
			gomega.Expect(requiredSortJoins("")).To(gomega.Equal(JoinType(0)))
		})
	})
})
