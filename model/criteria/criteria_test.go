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
				{ "isNot": { "genre": "Rock" }}
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
					`AND (not exists (select 1 from json_tree(participants, '$.artist') where key='name' and value = ?) ` +
					`OR media_file.album = ?) AND (media_file.comment LIKE ? AND (media_file.year >= ? AND media_file.year <= ?) ` +
					`AND not exists (select 1 from json_tree(tags, '$.genre') where key='value' and value = ?)))`))
			gomega.Expect(args).To(gomega.HaveExactElements("%love%", "%hate%", "u2", "best of", "this%", 1980, 1990, "Rock"))
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

			It("sorts by random", func() {
				newObj := goObj
				newObj.Sort = "random"
				gomega.Expect(newObj.OrderBy()).To(gomega.Equal("random() asc"))
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
				`(exists (select 1 from json_tree(participants, '$.artist') where key='name' and value = ?) AND ` +
					`exists (select 1 from json_tree(participants, '$.composer') where key='name' and value LIKE ?))`,
			))
			gomega.Expect(args).To(gomega.HaveExactElements("The Beatles", "%Lennon%"))
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
