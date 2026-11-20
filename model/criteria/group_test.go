package criteria

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Group criteria", func() {
	Describe("JSON", func() {
		It("round-trips a group with metadata", func() {
			input := `{"any":[
				{"all":[{"is":{"genre":"Rock"}}],"sort":"-rating","limit":20},
				{"any":[{"is":{"genre":"Jazz"}}],"limitPercent":10,"offset":2}
			],"sort":"title"}`

			var c Criteria
			gomega.Expect(json.Unmarshal([]byte(input), &c)).To(gomega.Succeed())

			anyGroup := c.Expression.(Any)
			first, ok := anyGroup[0].(Group)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(first.Sort).To(gomega.Equal("-rating"))
			gomega.Expect(first.Limit).To(gomega.Equal(20))

			second, ok := anyGroup[1].(Group)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(second.LimitPercent).To(gomega.Equal(10))
			gomega.Expect(second.Offset).To(gomega.Equal(2))

			out, err := json.Marshal(c)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(string(out)).To(gomega.MatchJSON(input))
		})

		It("keeps legacy groups as bare conjunctions", func() {
			var c Criteria
			gomega.Expect(json.Unmarshal([]byte(`{"all":[{"any":[{"is":{"loved":true}}]}]}`), &c)).To(gomega.Succeed())

			all := c.Expression.(All)
			_, ok := all[0].(Any)
			gomega.Expect(ok).To(gomega.BeTrue())
		})

		It("preserves empty groups with metadata", func() {
			input := `{"all":[{"all":[],"limit":0,"sort":""}]}`
			var c Criteria
			gomega.Expect(json.Unmarshal([]byte(input), &c)).To(gomega.Succeed())

			out, err := json.Marshal(c)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(string(out)).To(gomega.MatchJSON(`{"all":[{"all":[]}]}`))
		})

		It("accepts case-insensitive metadata keys", func() {
			var c Criteria
			gomega.Expect(json.Unmarshal([]byte(`{"all":[{"all":[],"LIMIT":3,"Sort":"title"}]}`), &c)).To(gomega.Succeed())
			g := c.Expression.(All)[0].(Group)
			gomega.Expect(g.Limit).To(gomega.Equal(3))
			gomega.Expect(g.Sort).To(gomega.Equal("title"))
		})

		It("keeps legacy nested all+any as sibling expressions", func() {
			var c Criteria
			input := `{"all":[{"all":[{"is":{"loved":true}}],"any":[{"is":{"genre":"Rock"}}]}]}`
			gomega.Expect(json.Unmarshal([]byte(input), &c)).To(gomega.Succeed())

			all := c.Expression.(All)
			gomega.Expect(all).To(gomega.HaveLen(2))
			gomega.Expect([]string{fmt.Sprintf("%T", all[0]), fmt.Sprintf("%T", all[1])}).
				To(gomega.ConsistOf("criteria.All", "criteria.Any"))
		})

		It("clamps group limitPercent", func() {
			var c Criteria
			gomega.Expect(json.Unmarshal([]byte(`{"all":[{"all":[],"limitPercent":150}]}`), &c)).To(gomega.Succeed())
			g := c.Expression.(All)[0].(Group)
			gomega.Expect(g.LimitPercent).To(gomega.Equal(100))

			gomega.Expect(json.Unmarshal([]byte(`{"all":[{"all":[],"limitPercent":-1}]}`), &c)).To(gomega.Succeed())
			g = c.Expression.(All)[0].(Group)
			gomega.Expect(g.LimitPercent).To(gomega.Equal(0))
		})

		DescribeTable("rejects invalid group JSON",
			func(input string) {
				var c Criteria
				gomega.Expect(json.Unmarshal([]byte(input), &c)).ToNot(gomega.Succeed())
			},
			Entry("both all and any with metadata", `{"all":[{"all":[],"any":[],"limit":1}]}`),
			Entry("missing conjunction with metadata", `{"all":[{"sort":"title"}]}`),
			Entry("unknown metadata key", `{"all":[{"all":[],"bogus":1}]}`),
			Entry("mixed operator and group keys", `{"all":[{"all":[],"is":{"title":"x"},"limit":1}]}`),
			Entry("negative limit", `{"all":[{"all":[],"limit":-1}]}`),
			Entry("negative offset", `{"all":[{"all":[],"offset":-1}]}`),
			Entry("non-numeric limit", `{"all":[{"all":[],"limit":"ten"}]}`),
		)
	})

	Describe("top-level Group marshaling", func() {
		It("flattens a non-empty group into criteria fields", func() {
			c := Criteria{Expression: Group{
				Expression: All{Is{"loved": true}},
				Sort:       "year",
				Limit:      5,
			}}
			out, err := json.Marshal(c)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(string(out)).To(gomega.MatchJSON(`{"all":[{"is":{"loved":true}}],"sort":"year","limit":5}`))

			var got Criteria
			gomega.Expect(json.Unmarshal(out, &got)).To(gomega.Succeed())
			gomega.Expect(got.Sort).To(gomega.Equal("year"))
			gomega.Expect(got.Limit).To(gomega.Equal(5))
		})

		It("errors instead of dropping an empty group's rules", func() {
			_, err := json.Marshal(Criteria{Expression: Group{Expression: All{}, Limit: 1}})
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		It("errors for a non-conjunction inner expression", func() {
			_, err := json.Marshal(Criteria{Expression: Group{Expression: Is{"loved": true}}})
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})

	Describe("selection helpers", func() {
		It("detects selection fields in nested groups", func() {
			c := Criteria{Expression: All{Group{Expression: All{}, LimitPercent: 10}}}
			gomega.Expect(c.HasNestedSelectionFields()).To(gomega.BeTrue())

			c = Criteria{Expression: All{Group{Expression: All{}, Sort: "title"}}}
			gomega.Expect(c.HasNestedSelectionFields()).To(gomega.BeFalse())
		})
	})

	Describe("child playlist extraction", func() {
		It("finds playlist references inside groups", func() {
			c := Criteria{Expression: All{
				Group{Expression: Any{
					InPlaylist{"id": "id-1"},
					NotInPlaylist{"path": "child.nsp"},
				}, Limit: 5},
			}}
			gomega.Expect(c.ChildPlaylistIds()).To(gomega.ConsistOf("id-1"))
			gomega.Expect(c.ChildPlaylistPaths()).To(gomega.ConsistOf("child.nsp"))
		})
	})

	Describe("sort inheritance", func() {
		parent := Criteria{Sort: "year", Order: "desc"}

		It("inherits the parent sort and order when the group has no sort", func() {
			g := Group{}
			gomega.Expect(g.OrderByFields(parent)).To(gomega.Equal([]SortField{{Field: "year", Desc: true}}))
		})

		It("does not inherit a parent order flip when the group has its own sort", func() {
			g := Group{Sort: "-rating"}
			gomega.Expect(g.OrderByFields(parent)).To(gomega.Equal([]SortField{{Field: "rating", Desc: true}}))
		})

		It("uses an explicit group order with its own sort", func() {
			g := Group{Sort: "rating", Order: "desc"}
			gomega.Expect(g.OrderByFields(parent)).To(gomega.Equal([]SortField{{Field: "rating", Desc: true}}))
		})
	})

	Describe("Walk", func() {
		It("visits groups and their children", func() {
			var visited []string
			err := Walk(All{Group{Expression: Any{Is{"loved": true}}, Limit: 1}}, func(expr Expression) error {
				visited = append(visited, fmt.Sprintf("%T", expr))
				return nil
			})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(visited).To(gomega.Equal([]string{"criteria.All", "criteria.Group", "criteria.Any", "criteria.Is"}))
		})
	})
})
