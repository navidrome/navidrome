package model_test

import (
	"bytes"
	"encoding/json"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SmartPlaylist", func() {
	var goObj model.SmartPlaylist
	var jsonObj string
	BeforeEach(func() {
		goObj = model.SmartPlaylist{
			RuleGroup: model.RuleGroup{
				Combinator: "and", Rules: model.Rules{
					model.Rule{Field: "title", Operator: "contains", Value: "love"},
					model.Rule{Field: "year", Operator: "is in the range", Value: []int{1980, 1989}},
					model.Rule{Field: "loved", Operator: "is true"},
					model.Rule{Field: "lastPlayed", Operator: "in the last", Value: 30},
					model.RuleGroup{
						Combinator: "or",
						Rules: model.Rules{
							model.Rule{Field: "artist", Operator: "is not", Value: "zé"},
							model.Rule{Field: "album", Operator: "is", Value: "4"},
						},
					},
				}},
			Order: "artist asc",
			Limit: 100,
		}
		var b bytes.Buffer
		err := json.Compact(&b, []byte(`
{
  "combinator":"and",
  "rules":[
    {
      "field":"title",
      "operator":"contains",
      "value":"love"
    },
    {
      "field":"year",
      "operator":"is in the range",
      "value":[
        1980,
        1989
      ]
    },
    {
      "field":"loved",
      "operator":"is true"
    },
    {
      "field":"lastPlayed",
      "operator":"in the last",
      "value":30
    },
    {
      "combinator":"or",
      "rules":[
        {
          "field":"artist",
          "operator":"is not",
          "value":"zé"
        },
        {
          "field":"album",
          "operator":"is",
          "value":"4"
        }
      ]
    }
  ],
  "order":"artist asc",
  "limit":100
}`))
		if err != nil {
			panic(err)
		}
		jsonObj = b.String()
	})
	It("finds all fields", func() {
		Expect(goObj.Fields()).To(ConsistOf("title", "year", "loved", "lastPlayed", "artist", "album"))
	})
	It("marshals to JSON", func() {
		j, err := json.Marshal(goObj)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(j)).To(Equal(jsonObj))
	})
	It("is reversible to/from JSON", func() {
		var newObj model.SmartPlaylist
		err := json.Unmarshal([]byte(jsonObj), &newObj)
		Expect(err).ToNot(HaveOccurred())
		j, err := json.Marshal(newObj)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(j)).To(Equal(jsonObj))
	})
})
