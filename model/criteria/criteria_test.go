package criteria

import (
	"bytes"
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Criteria", func() {
	var goObj Criteria
	var jsonObj string

	Context("with a complex criteria", func() {
		BeforeEach(func() {
			goObj = Criteria{
				Expression: All{
					Contains{"title": "love"},
					NotContains{"title": "hate"},
					Any{
						IsNot{"artist": "u2"},
						Is{"album": "best of"},
					},
					All{
						StartsWith{"comment": "this"},
						InTheRange{"year": []int{1980, 1990}},
						IsNot{"genre": "Rock"},
						Gt{"albumrating": 3},
					},
				},
				Sort:   "title",
				Order:  "asc",
				Limit:  20,
				Offset: 10,
			}
			var b bytes.Buffer
			err := json.Compact(&b, []byte(`
{
	"all": [
		{ "contains": {"title": "love"} },
		{ "notContains": {"title": "hate"} },
		{ "any": [
				{ "isNot": {"artist": "u2"} },
				{ "is": {"album": "best of"} }
			]
		},
		{ "all": [
				{ "startsWith": {"comment": "this"} },
				{ "inTheRange": {"year":[1980,1990]} },
				{ "isNot": { "genre": "Rock" }},
				{ "gt": { "albumrating": 3 } }
			]
		}
	],
	"sort": "title",
	"order": "asc",
	"limit": 20,
	"offset": 10
}
`))
			if err != nil {
				panic(err)
			}
			jsonObj = b.String()
		})
		It("marshals to JSON", func() {
			j, err := json.Marshal(goObj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(string(j)).To(gomega.Equal(jsonObj))
		})
		It("is reversible to/from JSON", func() {
			var newObj Criteria
			err := json.Unmarshal([]byte(jsonObj), &newObj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			j, err := json.Marshal(newObj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(string(j)).To(gomega.Equal(jsonObj))
		})
	})

	Describe("LimitPercent", func() {
		Describe("JSON round-trip", func() {
			It("marshals and unmarshals limitPercent", func() {
				goObj := Criteria{
					Expression:   All{Contains{"title": "love"}},
					Sort:         "title",
					Order:        "asc",
					LimitPercent: 10,
				}
				j, err := json.Marshal(goObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(string(j)).To(gomega.ContainSubstring(`"limitPercent":10`))
				gomega.Expect(string(j)).ToNot(gomega.ContainSubstring(`"limit"`))

				var newObj Criteria
				err = json.Unmarshal(j, &newObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(newObj.LimitPercent).To(gomega.Equal(10))
				gomega.Expect(newObj.Limit).To(gomega.Equal(0))
			})

			It("does not include limitPercent when zero", func() {
				goObj := Criteria{
					Expression: All{Contains{"title": "love"}},
					Limit:      50,
				}
				j, err := json.Marshal(goObj)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(string(j)).To(gomega.ContainSubstring(`"limit":50`))
				gomega.Expect(string(j)).ToNot(gomega.ContainSubstring(`limitPercent`))
			})

			It("backward compatible: JSON with only limit still works", func() {
				jsonStr := `{"all":[{"contains":{"title":"love"}}],"limit":20}`
				var c Criteria
				err := json.Unmarshal([]byte(jsonStr), &c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(c.Limit).To(gomega.Equal(20))
				gomega.Expect(c.LimitPercent).To(gomega.Equal(0))
			})
		})

		Describe("UnmarshalJSON clamping", func() {
			It("clamps values above 100 to 100", func() {
				jsonStr := `{"all":[{"contains":{"title":"love"}}],"limitPercent":150}`
				var c Criteria
				err := json.Unmarshal([]byte(jsonStr), &c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(c.LimitPercent).To(gomega.Equal(100))
			})

			It("clamps negative values to 0", func() {
				jsonStr := `{"all":[{"contains":{"title":"love"}}],"limitPercent":-5}`
				var c Criteria
				err := json.Unmarshal([]byte(jsonStr), &c)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(c.LimitPercent).To(gomega.Equal(0))
			})
		})

		Describe("EffectiveLimit", func() {
			It("returns fixed limit when Limit is set", func() {
				c := Criteria{Limit: 50, LimitPercent: 10}
				gomega.Expect(c.EffectiveLimit(1000)).To(gomega.Equal(50))
			})

			It("returns percentage-based limit", func() {
				c := Criteria{LimitPercent: 10}
				gomega.Expect(c.EffectiveLimit(450)).To(gomega.Equal(45))
			})

			It("returns minimum 1 when totalCount > 0 and percentage rounds to 0", func() {
				c := Criteria{LimitPercent: 1}
				gomega.Expect(c.EffectiveLimit(5)).To(gomega.Equal(1))
			})

			It("returns 0 when totalCount is 0", func() {
				c := Criteria{LimitPercent: 10}
				gomega.Expect(c.EffectiveLimit(0)).To(gomega.Equal(0))
			})

			It("returns 0 when no limit is set", func() {
				c := Criteria{}
				gomega.Expect(c.EffectiveLimit(1000)).To(gomega.Equal(0))
			})

			It("returns full count for 100%", func() {
				c := Criteria{LimitPercent: 100}
				gomega.Expect(c.EffectiveLimit(450)).To(gomega.Equal(450))
			})

			It("returns 1 for 1% of 50 items", func() {
				c := Criteria{LimitPercent: 1}
				gomega.Expect(c.EffectiveLimit(50)).To(gomega.Equal(1))
			})
		})

		Describe("IsPercentageLimit", func() {
			It("returns true when LimitPercent is set and Limit is 0", func() {
				c := Criteria{LimitPercent: 10}
				gomega.Expect(c.IsPercentageLimit()).To(gomega.BeTrue())
			})

			It("returns false when Limit is set", func() {
				c := Criteria{Limit: 50, LimitPercent: 10}
				gomega.Expect(c.IsPercentageLimit()).To(gomega.BeFalse())
			})

			It("returns false when neither is set", func() {
				c := Criteria{}
				gomega.Expect(c.IsPercentageLimit()).To(gomega.BeFalse())
			})

			It("returns false when LimitPercent is out of range", func() {
				c := Criteria{LimitPercent: 150}
				gomega.Expect(c.IsPercentageLimit()).To(gomega.BeFalse())
			})
		})
	})

	Context("with child playlists", func() {
		var (
			topLevelInPlaylistID     string
			topLevelNotInPlaylistID  string
			nestedAnyInPlaylistID    string
			nestedAnyNotInPlaylistID string
			nestedAllInPlaylistID    string
			nestedAllNotInPlaylistID string
		)
		BeforeEach(func() {
			topLevelInPlaylistID = uuid.NewString()
			topLevelNotInPlaylistID = uuid.NewString()

			nestedAnyInPlaylistID = uuid.NewString()
			nestedAnyNotInPlaylistID = uuid.NewString()

			nestedAllInPlaylistID = uuid.NewString()
			nestedAllNotInPlaylistID = uuid.NewString()

			goObj = Criteria{
				Expression: All{
					InPlaylist{"id": topLevelInPlaylistID},
					NotInPlaylist{"id": topLevelNotInPlaylistID},
					Any{
						InPlaylist{"id": nestedAnyInPlaylistID},
						NotInPlaylist{"id": nestedAnyNotInPlaylistID},
					},
					All{
						InPlaylist{"id": nestedAllInPlaylistID},
						NotInPlaylist{"id": nestedAllNotInPlaylistID},
					},
				},
			}
		})
		It("extracts all child smart playlist IDs from expression criteria", func() {
			ids := goObj.ChildPlaylistIds()
			gomega.Expect(ids).To(gomega.ConsistOf(topLevelInPlaylistID, topLevelNotInPlaylistID, nestedAnyInPlaylistID, nestedAnyNotInPlaylistID, nestedAllInPlaylistID, nestedAllNotInPlaylistID))
		})
		It("extracts child smart playlist IDs from deeply nested expression", func() {
			goObj = Criteria{
				Expression: Any{
					Any{
						All{
							Any{
								InPlaylist{"id": nestedAnyInPlaylistID},
								NotInPlaylist{"id": nestedAnyNotInPlaylistID},
								Any{
									All{
										InPlaylist{"id": nestedAllInPlaylistID},
										NotInPlaylist{"id": nestedAllNotInPlaylistID},
									},
								},
							},
						},
					},
				},
			}

			ids := goObj.ChildPlaylistIds()
			gomega.Expect(ids).To(gomega.ConsistOf(nestedAnyInPlaylistID, nestedAnyNotInPlaylistID, nestedAllInPlaylistID, nestedAllNotInPlaylistID))
		})
		It("returns empty list when no child playlist IDs are present", func() {
			ids := Criteria{}.ChildPlaylistIds()
			gomega.Expect(ids).To(gomega.BeEmpty())
		})
		It("returns empty list for leaf expressions", func() {
			ids := Criteria{Expression: Is{"title": "Low Rider"}}.ChildPlaylistIds()
			gomega.Expect(ids).To(gomega.BeEmpty())
		})
	})
})
