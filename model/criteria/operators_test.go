package criteria_test

import (
	"encoding/json"
	"fmt"

	. "github.com/navidrome/navidrome/model/criteria"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	AddRoles([]string{"artist", "composer"})
	AddTagNames([]string{"genre"})
	AddNumericTags([]string{"rate"})
})

var _ = Describe("Operators", func() {
	DescribeTable("JSON Marshaling",
		func(op Expression, jsonString string) {
			obj := And{op}
			newJs, err := json.Marshal(obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(string(newJs)).To(gomega.Equal(fmt.Sprintf(`{"all":[%s]}`, jsonString)))

			var unmarshalObj UnmarshalConjunctionType
			js := "[" + jsonString + "]"
			err = json.Unmarshal([]byte(js), &unmarshalObj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(unmarshalObj[0]).To(gomega.Equal(op))
		},
		Entry("is [string]", Is{"title": "Low Rider"}, `{"is":{"title":"Low Rider"}}`),
		Entry("is [bool]", Is{"loved": false}, `{"is":{"loved":false}}`),
		Entry("isNot", IsNot{"title": "Low Rider"}, `{"isNot":{"title":"Low Rider"}}`),
		Entry("gt", Gt{"playCount": 10.0}, `{"gt":{"playCount":10}}`),
		Entry("lt", Lt{"playCount": 10.0}, `{"lt":{"playCount":10}}`),
		Entry("contains", Contains{"title": "Low Rider"}, `{"contains":{"title":"Low Rider"}}`),
		Entry("notContains", NotContains{"title": "Low Rider"}, `{"notContains":{"title":"Low Rider"}}`),
		Entry("startsWith", StartsWith{"title": "Low Rider"}, `{"startsWith":{"title":"Low Rider"}}`),
		Entry("endsWith", EndsWith{"title": "Low Rider"}, `{"endsWith":{"title":"Low Rider"}}`),
		Entry("inTheRange [number]", InTheRange{"year": []any{1980.0, 1990.0}}, `{"inTheRange":{"year":[1980,1990]}}`),
		Entry("inTheRange [date]", InTheRange{"lastPlayed": []any{"2021-10-01", "2021-11-01"}}, `{"inTheRange":{"lastPlayed":["2021-10-01","2021-11-01"]}}`),
		Entry("before", Before{"lastPlayed": "2021-10-01"}, `{"before":{"lastPlayed":"2021-10-01"}}`),
		Entry("after", After{"lastPlayed": "2021-10-01"}, `{"after":{"lastPlayed":"2021-10-01"}}`),
		Entry("inTheLast", InTheLast{"lastPlayed": 30.0}, `{"inTheLast":{"lastPlayed":30}}`),
		Entry("notInTheLast", NotInTheLast{"lastPlayed": 30.0}, `{"notInTheLast":{"lastPlayed":30}}`),
		Entry("inPlaylist", InPlaylist{"id": "deadbeef-dead-beef"}, `{"inPlaylist":{"id":"deadbeef-dead-beef"}}`),
		Entry("notInPlaylist", NotInPlaylist{"id": "deadbeef-dead-beef"}, `{"notInPlaylist":{"id":"deadbeef-dead-beef"}}`),
	)
})
