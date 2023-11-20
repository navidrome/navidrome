package criteria

import (
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Criteria", func() {
	var goObj Criteria
	var jsonObj string
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
					IsNot{"genre": "test"},
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
				{ "isNot": { "genre": "test" }}
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
		gomega.Expect(sql).To(gomega.Equal("(media_file.title LIKE ? AND media_file.title NOT LIKE ? AND (media_file.artist <> ? OR media_file.album = ?) AND (media_file.comment LIKE ? AND (media_file.year >= ? AND media_file.year <= ?) AND COALESCE(genre.name, '') <> ?))"))
		gomega.Expect(args).To(gomega.ConsistOf("%love%", "%hate%", "u2", "best of", "this%", 1980, 1990, "test"))
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

	It("allows sort by random", func() {
		newObj := goObj
		newObj.Sort = "random"
		gomega.Expect(newObj.OrderBy()).To(gomega.Equal("random() asc"))
	})

})
