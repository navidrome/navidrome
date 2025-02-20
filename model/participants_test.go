package model_test

import (
	"encoding/json"

	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Participants", func() {
	Describe("JSON Marshalling", func() {
		When("we have a valid Albums object", func() {
			var participants Participants
			BeforeEach(func() {
				participants = Participants{
					RoleArtist:      []Participant{_p("1", "Artist1"), _p("2", "Artist2")},
					RoleAlbumArtist: []Participant{_p("3", "AlbumArtist1"), _p("4", "AlbumArtist2")},
				}
			})

			It("marshals correctly", func() {
				data, err := json.Marshal(participants)
				Expect(err).To(BeNil())

				var afterConversion Participants
				err = json.Unmarshal(data, &afterConversion)
				Expect(err).To(BeNil())
				Expect(afterConversion).To(Equal(participants))
			})

			It("returns unmarshal error when the role is invalid", func() {
				err := json.Unmarshal([]byte(`{"unknown": []}`), &participants)
				Expect(err).To(MatchError("invalid role: unknown"))
			})
		})
	})

	Describe("First", func() {
		var participants Participants
		BeforeEach(func() {
			participants = Participants{
				RoleArtist:      []Participant{_p("1", "Artist1"), _p("2", "Artist2")},
				RoleAlbumArtist: []Participant{_p("3", "AlbumArtist1"), _p("4", "AlbumArtist2")},
			}
		})
		It("returns the first artist of the role", func() {
			Expect(participants.First(RoleArtist)).To(Equal(Artist{ID: "1", Name: "Artist1"}))
		})
		It("returns an empty artist when the role is not present", func() {
			Expect(participants.First(RoleComposer)).To(Equal(Artist{}))
		})
	})

	Describe("Add", func() {
		var participants Participants
		BeforeEach(func() {
			participants = Participants{
				RoleArtist: []Participant{_p("1", "Artist1"), _p("2", "Artist2")},
			}
		})
		It("adds the artist to the role", func() {
			participants.Add(RoleArtist, Artist{ID: "5", Name: "Artist5"})
			Expect(participants).To(Equal(Participants{
				RoleArtist: []Participant{_p("1", "Artist1"), _p("2", "Artist2"), _p("5", "Artist5")},
			}))
		})
		It("creates a new role if it doesn't exist", func() {
			participants.Add(RoleComposer, Artist{ID: "5", Name: "Artist5"})
			Expect(participants).To(Equal(Participants{
				RoleArtist:   []Participant{_p("1", "Artist1"), _p("2", "Artist2")},
				RoleComposer: []Participant{_p("5", "Artist5")},
			}))
		})
		It("should not add duplicate artists", func() {
			participants.Add(RoleArtist, Artist{ID: "1", Name: "Artist1"})
			Expect(participants).To(Equal(Participants{
				RoleArtist: []Participant{_p("1", "Artist1"), _p("2", "Artist2")},
			}))
		})
		It("adds the artist with and without subrole", func() {
			participants = Participants{}
			participants.Add(RolePerformer, Artist{ID: "3", Name: "Artist3"})
			participants.AddWithSubRole(RolePerformer, "SubRole", Artist{ID: "3", Name: "Artist3"})

			artist3 := _p("3", "Artist3")
			artist3WithSubRole := artist3
			artist3WithSubRole.SubRole = "SubRole"

			Expect(participants[RolePerformer]).To(HaveLen(2))
			Expect(participants).To(Equal(Participants{
				RolePerformer: []Participant{
					artist3,
					artist3WithSubRole,
				},
			}))
		})
	})

	Describe("Merge", func() {
		var participations1, participations2 Participants
		BeforeEach(func() {
			participations1 = Participants{
				RoleArtist:      []Participant{_p("1", "Artist1"), _p("2", "Duplicated Artist")},
				RoleAlbumArtist: []Participant{_p("3", "AlbumArtist1"), _p("4", "AlbumArtist2")},
			}
			participations2 = Participants{
				RoleArtist:      []Participant{_p("5", "Artist3"), _p("6", "Artist4"), _p("2", "Duplicated Artist")},
				RoleAlbumArtist: []Participant{_p("7", "AlbumArtist3"), _p("8", "AlbumArtist4")},
			}
		})
		It("merges correctly, skipping duplicated artists", func() {
			participations1.Merge(participations2)
			Expect(participations1).To(Equal(Participants{
				RoleArtist:      []Participant{_p("1", "Artist1"), _p("2", "Duplicated Artist"), _p("5", "Artist3"), _p("6", "Artist4")},
				RoleAlbumArtist: []Participant{_p("3", "AlbumArtist1"), _p("4", "AlbumArtist2"), _p("7", "AlbumArtist3"), _p("8", "AlbumArtist4")},
			}))
		})
	})

	Describe("Hash", func() {
		It("should return the same hash for the same participants", func() {
			p1 := Participants{
				RoleArtist:      []Participant{_p("1", "Artist1"), _p("2", "Artist2")},
				RoleAlbumArtist: []Participant{_p("3", "AlbumArtist1"), _p("4", "AlbumArtist2")},
			}
			p2 := Participants{
				RoleArtist:      []Participant{_p("2", "Artist2"), _p("1", "Artist1")},
				RoleAlbumArtist: []Participant{_p("4", "AlbumArtist2"), _p("3", "AlbumArtist1")},
			}
			Expect(p1.Hash()).To(Equal(p2.Hash()))
		})
		It("should return different hashes for different participants", func() {
			p1 := Participants{
				RoleArtist: []Participant{_p("1", "Artist1")},
			}
			p2 := Participants{
				RoleArtist: []Participant{_p("1", "Artist1"), _p("2", "Artist2")},
			}
			Expect(p1.Hash()).ToNot(Equal(p2.Hash()))
		})
	})

	Describe("All", func() {
		var participants Participants
		BeforeEach(func() {
			participants = Participants{
				RoleArtist:      []Participant{_p("1", "Artist1"), _p("2", "Artist2")},
				RoleAlbumArtist: []Participant{_p("3", "AlbumArtist1"), _p("4", "AlbumArtist2")},
				RoleProducer:    []Participant{_p("5", "Producer", "SortProducerName")},
				RoleComposer:    []Participant{_p("1", "Artist1")},
			}
		})

		Describe("All", func() {
			It("returns all artists found in the Participants", func() {
				artists := participants.AllArtists()
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
			It("returns all artist IDs found in the Participants", func() {
				ids := participants.AllIDs()
				Expect(ids).To(ConsistOf("1", "2", "3", "4", "5"))
			})
		})

		Describe("AllNames", func() {
			It("returns all artist names found in the Participants", func() {
				names := participants.AllNames()
				Expect(names).To(ConsistOf("Artist1", "Artist2", "AlbumArtist1", "AlbumArtist2",
					"Producer", "SortProducerName"))
			})
		})
	})
})

var _ = Describe("ParticipantList", func() {
	Describe("Join", func() {
		It("joins the participants with the given separator", func() {
			list := ParticipantList{
				_p("1", "Artist 1"),
				_p("3", "Artist 2"),
			}
			list[0].SubRole = "SubRole"
			Expect(list.Join(", ")).To(Equal("Artist 1 (SubRole), Artist 2"))
		})

		It("returns the sole participant if there is only one", func() {
			list := ParticipantList{_p("1", "Artist 1")}
			Expect(list.Join(", ")).To(Equal("Artist 1"))
		})

		It("returns empty string if there are no participants", func() {
			var list ParticipantList
			Expect(list.Join(", ")).To(Equal(""))
		})
	})
})

func _p(id, name string, sortName ...string) Participant {
	p := Participant{Artist: Artist{ID: id, Name: name}}
	if len(sortName) > 0 {
		p.Artist.SortArtistName = sortName[0]
	}
	return p
}
