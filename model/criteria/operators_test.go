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
		Entry("is [string does not coerce non-boolean field]", Is{"title": "true"}, `{"is":{"title":"true"}}`),
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
		Entry("isMissing [true]", IsMissing{"genre": true}, `{"isMissing":{"genre":true}}`),
		Entry("isMissing [false]", IsMissing{"genre": false}, `{"isMissing":{"genre":false}}`),
		Entry("isPresent [true]", IsPresent{"genre": true}, `{"isPresent":{"genre":true}}`),
		Entry("isPresent [false]", IsPresent{"genre": false}, `{"isPresent":{"genre":false}}`),
	)

	Describe("Boolean string coercion at unmarshal time (issue #4826)", func() {
		It("coerces string 'true' to bool for boolean fields", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"is":{"loved":"true"}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(Is{"loved": true}))
		})

		It("coerces string 'false' to bool for boolean fields", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"is":{"loved":"false"}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(Is{"loved": false}))
		})

		It("does not coerce string values for non-boolean fields", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"is":{"title":"true"}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(Is{"title": "true"}))
		})

		It("coerces numeric 1 to bool true for boolean fields", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"is":{"loved":1}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(Is{"loved": true}))
		})

		It("coerces numeric 0 to bool false for boolean fields", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"is":{"loved":0}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(Is{"loved": false}))
		})

		It("coerces in nested any/all groups", func() {
			var c Criteria
			err := json.Unmarshal([]byte(`{"all":[{"contains":{"title":"love"}},{"any":[{"is":{"loved":"true"}}]}]}`), &c)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			all := c.Expression.(All)
			nested := all[1].(Any)
			gomega.Expect(nested[0]).To(gomega.Equal(Is{"loved": true}))
		})

		It("coerces isMissing string 'true' to bool", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"isMissing":{"genre":"true"}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(IsMissing{"genre": true}))
		})

		It("coerces isMissing numeric 0 to bool false", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"isMissing":{"genre":0}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(IsMissing{"genre": false}))
		})

		It("coerces isPresent string 'false' to bool", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"isPresent":{"genre":"false"}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(IsPresent{"genre": false}))
		})

		It("coerces isPresent numeric 1 to bool true", func() {
			var obj UnmarshalConjunctionType
			err := json.Unmarshal([]byte(`[{"isPresent":{"genre":1}}]`), &obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(obj[0]).To(gomega.Equal(IsPresent{"genre": true}))
		})
	})
})
