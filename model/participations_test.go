package model_test

import (
	"encoding/json"

	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Participations", func() {
	Describe("JSON Marshalling", func() {
		When("we have a valid Albums object", func() {
			var participations Participations
			BeforeEach(func() {
				participations = Participations{
					RoleArtist:      []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Artist2"}},
					RoleAlbumArtist: []Artist{{ID: "3", Name: "AlbumArtist1"}, {ID: "4", Name: "AlbumArtist2"}},
				}
			})

			It("marshals correctly", func() {
				data, err := json.Marshal(participations)
				Expect(err).To(BeNil())

				var afterConversion Participations
				err = json.Unmarshal(data, &afterConversion)
				Expect(err).To(BeNil())
				Expect(afterConversion).To(Equal(participations))
			})

			It("returns unmarshal error when the role is invalid", func() {
				err := json.Unmarshal([]byte(`{"unknown": []}`), &participations)
				Expect(err).To(MatchError("invalid role: unknown"))
			})
		})
	})

	Describe("First", func() {
		var participations Participations
		BeforeEach(func() {
			participations = Participations{
				RoleArtist:      []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Artist2"}},
				RoleAlbumArtist: []Artist{{ID: "3", Name: "AlbumArtist1"}, {ID: "4", Name: "AlbumArtist2"}},
			}
		})
		It("returns the first artist of the role", func() {
			Expect(participations.First(RoleArtist)).To(Equal(Artist{ID: "1", Name: "Artist1"}))
		})
		It("returns an empty artist when the role is not present", func() {
			Expect(participations.First(RoleComposer)).To(Equal(Artist{}))
		})
	})

	Describe("Add", func() {
		var participations Participations
		BeforeEach(func() {
			participations = Participations{
				RoleArtist: []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Artist2"}},
			}
		})
		It("adds the artist to the role", func() {
			participations.Add(RoleArtist, Artist{ID: "5", Name: "Artist3"})
			Expect(participations).To(Equal(Participations{
				RoleArtist: []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Artist2"}, {ID: "5", Name: "Artist3"}},
			}))
		})
		It("creates a new role if it doesn't exist", func() {
			participations.Add(RoleComposer, Artist{ID: "5", Name: "Artist3"})
			Expect(participations).To(Equal(Participations{
				RoleArtist:   []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Artist2"}},
				RoleComposer: []Artist{{ID: "5", Name: "Artist3"}},
			}))
		})
		It("should not add duplicate artists", func() {
			participations.Add(RoleArtist, Artist{ID: "1", Name: "Artist1"})
			Expect(participations).To(Equal(Participations{
				RoleArtist: []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Artist2"}},
			}))
		})
	})

	Describe("Merge", func() {
		var participations1, participations2 Participations
		BeforeEach(func() {
			participations1 = Participations{
				RoleArtist:      []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Duplicated Artist"}},
				RoleAlbumArtist: []Artist{{ID: "3", Name: "AlbumArtist1"}, {ID: "4", Name: "AlbumArtist2"}},
			}
			participations2 = Participations{
				RoleArtist:      []Artist{{ID: "5", Name: "Artist3"}, {ID: "6", Name: "Artist4"}, {ID: "2", Name: "Duplicated Artist"}},
				RoleAlbumArtist: []Artist{{ID: "7", Name: "AlbumArtist3"}, {ID: "8", Name: "AlbumArtist4"}},
			}
		})
		It("merges correctly, skipping duplicated artists", func() {
			participations1.Merge(participations2)
			Expect(participations1).To(Equal(Participations{
				RoleArtist:      []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Duplicated Artist"}, {ID: "5", Name: "Artist3"}, {ID: "6", Name: "Artist4"}},
				RoleAlbumArtist: []Artist{{ID: "3", Name: "AlbumArtist1"}, {ID: "4", Name: "AlbumArtist2"}, {ID: "7", Name: "AlbumArtist3"}, {ID: "8", Name: "AlbumArtist4"}},
			}))
		})
	})

	Describe("All", func() {
		var participations Participations
		BeforeEach(func() {
			participations = Participations{
				RoleArtist:      []Artist{{ID: "1", Name: "Artist1"}, {ID: "2", Name: "Artist2"}},
				RoleAlbumArtist: []Artist{{ID: "3", Name: "AlbumArtist1"}, {ID: "4", Name: "AlbumArtist2"}},
				RoleProducer:    []Artist{{ID: "5", Name: "Producer", SortArtistName: "SortProducerName"}},
				RoleComposer:    []Artist{{ID: "1", Name: "Artist1"}},
			}
		})

		Describe("All", func() {
			It("returns all artists found in the Participations", func() {
				artists := participations.All()
				Expect(artists).To(ConsistOf(
					Artist{ID: "1", Name: "Artist1"},
					Artist{ID: "2", Name: "Artist2"},
					Artist{ID: "3", Name: "AlbumArtist1"},
					Artist{ID: "4", Name: "AlbumArtist2"},
					Artist{ID: "5", Name: "Producer", SortArtistName: "SortProducerName"},
				))
			})
		})

		Describe("AllIDs", func() {
			It("returns all artist IDs found in the Participations", func() {
				ids := participations.AllIDs()
				Expect(ids).To(ConsistOf("1", "2", "3", "4", "5"))
			})
		})

		Describe("AllNames", func() {
			It("returns all artist names found in the Participations", func() {
				names := participations.AllNames()
				Expect(names).To(ConsistOf("Artist1", "Artist2", "AlbumArtist1", "AlbumArtist2",
					"Producer", "SortProducerName"))
			})
		})
	})
})
