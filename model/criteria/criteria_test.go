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
		It("generates valid SQL", func() {
			sql, args, err := goObj.ToSql()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(sql).To(gomega.Equal(
				`(media_file.title LIKE ? AND media_file.title NOT LIKE ? ` +
					`AND (not exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name' and value = ?) ` +
					`OR media_file.album = ?) AND (media_file.comment LIKE ? AND (media_file.year >= ? AND media_file.year <= ?) ` +
					`AND not exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value' and value = ?) AND COALESCE(album_annotation.rating, 0) > ?))`))
			gomega.Expect(args).To(gomega.HaveExactElements("%love%", "%hate%", "u2", "best of", "this%", 1980, 1990, "Rock", 3))
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
		Describe("OrderBy", func() {
			It("sorts by regular fields", func() {
				gomega.Expect(goObj.OrderBy()).To(gomega.Equal("media_file.title asc"))
			})

			It("sorts by tag fields", func() {
				goObj.Sort = "genre"
				gomega.Expect(goObj.OrderBy()).To(
					gomega.Equal(
						"COALESCE(json_extract(media_file.tags, '$.genre[0].value'), '') asc",
					),
				)
			})

			It("sorts by role fields", func() {
				goObj.Sort = "artist"
				gomega.Expect(goObj.OrderBy()).To(
					gomega.Equal(
						"COALESCE(json_extract(media_file.participants, '$.artist[0].name'), '') asc",
					),
				)
			})

			It("casts numeric tags when sorting", func() {
				AddTagNames([]string{"rate"})
				AddNumericTags([]string{"rate"})
				goObj.Sort = "rate"
				gomega.Expect(goObj.OrderBy()).To(
					gomega.Equal("CAST(COALESCE(json_extract(media_file.tags, '$.rate[0].value'), '') AS REAL) asc"),
				)
			})

			It("sorts by albumtype alias (resolves to releasetype)", func() {
				AddTagNames([]string{"releasetype"})
				goObj.Sort = "albumtype"
				gomega.Expect(goObj.OrderBy()).To(
					gomega.Equal(
						"COALESCE(json_extract(media_file.tags, '$.releasetype[0].value'), '') asc",
					),
				)
			})

			It("sorts by random", func() {
				newObj := goObj
				newObj.Sort = "random"
				gomega.Expect(newObj.OrderBy()).To(gomega.Equal("random() asc"))
			})

			It("sorts by multiple fields", func() {
				goObj.Sort = "title,-rating"
				gomega.Expect(goObj.OrderBy()).To(gomega.Equal(
					"media_file.title asc, COALESCE(annotation.rating, 0) desc",
				))
			})

			It("reverts order when order is desc", func() {
				goObj.Sort = "-date,artist"
				goObj.Order = "desc"
				gomega.Expect(goObj.OrderBy()).To(gomega.Equal(
					"media_file.date asc, COALESCE(json_extract(media_file.participants, '$.artist[0].name'), '') desc",
				))
			})

			It("ignores invalid sort fields", func() {
				goObj.Sort = "bogus,title"
				gomega.Expect(goObj.OrderBy()).To(gomega.Equal(
					"media_file.title asc",
				))
			})
		})
	})

	Context("with artist roles", func() {
		BeforeEach(func() {
			goObj = Criteria{
				Expression: All{
					Is{"artist": "The Beatles"},
					Contains{"composer": "Lennon"},
				},
			}
		})

		It("generates valid SQL", func() {
			sql, args, err := goObj.ToSql()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(sql).To(gomega.Equal(
				`(exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name' and value = ?) AND ` +
					`exists (select 1 from json_tree(media_file.participants, '$.composer') where key='name' and value LIKE ?))`,
			))
			gomega.Expect(args).To(gomega.HaveExactElements("The Beatles", "%Lennon%"))
		})
	})

	Describe("ExpressionJoins", func() {
		It("excludes sort-only joins", func() {
			c := Criteria{
				Expression: All{
					Contains{"title": "love"},
				},
				Sort: "albumRating",
			}
			gomega.Expect(c.ExpressionJoins()).To(gomega.Equal(JoinNone))
			gomega.Expect(c.RequiredJoins().Has(JoinAlbumAnnotation)).To(gomega.BeTrue())
		})

		It("includes expression-based joins", func() {
			c := Criteria{
				Expression: All{
					Gt{"albumRating": 3},
				},
			}
			gomega.Expect(c.ExpressionJoins().Has(JoinAlbumAnnotation)).To(gomega.BeTrue())
		})
	})

	Describe("RequiredJoins", func() {
		It("returns JoinNone when no annotation fields are used", func() {
			c := Criteria{
				Expression: All{
					Contains{"title": "love"},
				},
			}
			gomega.Expect(c.RequiredJoins()).To(gomega.Equal(JoinNone))
		})
		It("returns JoinNone for media_file annotation fields", func() {
			c := Criteria{
				Expression: All{
					Is{"loved": true},
					Gt{"playCount": 5},
				},
			}
			gomega.Expect(c.RequiredJoins()).To(gomega.Equal(JoinNone))
		})
		It("returns JoinAlbumAnnotation for album annotation fields", func() {
			c := Criteria{
				Expression: All{
					Gt{"albumRating": 3},
				},
			}
			gomega.Expect(c.RequiredJoins()).To(gomega.Equal(JoinAlbumAnnotation))
		})
		It("returns JoinArtistAnnotation for artist annotation fields", func() {
			c := Criteria{
				Expression: All{
					Is{"artistLoved": true},
				},
			}
			gomega.Expect(c.RequiredJoins()).To(gomega.Equal(JoinArtistAnnotation))
		})
		It("returns both join types when both are used", func() {
			c := Criteria{
				Expression: All{
					Gt{"albumRating": 3},
					Is{"artistLoved": true},
				},
			}
			j := c.RequiredJoins()
			gomega.Expect(j.Has(JoinAlbumAnnotation)).To(gomega.BeTrue())
			gomega.Expect(j.Has(JoinArtistAnnotation)).To(gomega.BeTrue())
		})
		It("detects join types in nested expressions", func() {
			c := Criteria{
				Expression: All{
					Any{
						All{
							Is{"albumLoved": true},
						},
					},
					Any{
						Gt{"artistPlayCount": 10},
					},
				},
			}
			j := c.RequiredJoins()
			gomega.Expect(j.Has(JoinAlbumAnnotation)).To(gomega.BeTrue())
			gomega.Expect(j.Has(JoinArtistAnnotation)).To(gomega.BeTrue())
		})
		It("detects join types from Sort field", func() {
			c := Criteria{
				Expression: All{
					Contains{"title": "love"},
				},
				Sort: "albumRating",
			}
			gomega.Expect(c.RequiredJoins().Has(JoinAlbumAnnotation)).To(gomega.BeTrue())
		})
		It("detects join types from Sort field with direction prefix", func() {
			c := Criteria{
				Expression: All{
					Contains{"title": "love"},
				},
				Sort: "-artistRating",
			}
			gomega.Expect(c.RequiredJoins().Has(JoinArtistAnnotation)).To(gomega.BeTrue())
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
	})
})
