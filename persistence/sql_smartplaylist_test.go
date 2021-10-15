package persistence

import (
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("SmartPlaylist", func() {
	var pls SmartPlaylist
	Describe("AddFilters", func() {
		BeforeEach(func() {
			sp := model.SmartPlaylist{
				RuleGroup: model.RuleGroup{
					Combinator: "and", Rules: model.Rules{
						model.Rule{Field: "title", Operator: "contains", Value: "love"},
						model.Rule{Field: "year", Operator: "is in the range", Value: []int{1980, 1989}},
						model.Rule{Field: "loved", Operator: "is true"},
						model.Rule{Field: "lastPlayed", Operator: "in the last", Value: "30"},
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
			pls = SmartPlaylist(sp)
		})

		It("returns a proper SQL query", func() {
			sel := pls.AddFilters(squirrel.Select("media_file").Columns("*"))
			sql, args, err := sel.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(Equal("SELECT media_file, * WHERE (media_file.title ILIKE ? AND (media_file.year >= ? AND media_file.year <= ?) AND annotation.starred = ? AND annotation.play_date > ? AND (media_file.artist <> ? OR media_file.album = ?)) ORDER BY artist asc LIMIT 100"))
			lastMonth := time.Now().Add(-30 * 24 * time.Hour)
			Expect(args).To(ConsistOf("%love%", 1980, 1989, true, BeTemporally("~", lastMonth, time.Second), "zé", "4"))
		})
		It("returns an error if field is invalid", func() {
			r := pls.Rules[0].(model.Rule)
			r.Field = "INVALID"
			pls.Rules[0] = r
			sel := pls.AddFilters(squirrel.Select("media_file").Columns("*"))
			_, _, err := sel.ToSql()
			Expect(err).To(MatchError("invalid smart playlist field 'INVALID'"))
		})
	})

	Describe("fieldMap", func() {
		It("includes all possible fields", func() {
			for _, field := range model.SmartPlaylistFields {
				Expect(fieldMap).To(HaveKey(field))
			}
		})
		It("does not have extra fields", func() {
			for field := range fieldMap {
				Expect(model.SmartPlaylistFields).To(ContainElement(field))
			}
		})
	})

	Describe("stringRule", func() {
		DescribeTable("stringRule",
			func(operator, expectedSql, expectedValue string) {
				r := stringRule{Field: "title", Operator: operator, Value: "value"}
				sql, args, err := r.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(expectedSql))
				Expect(args).To(ConsistOf(expectedValue))
			},
			Entry("is", "is", "title = ?", "value"),
			Entry("is not", "is not", "title <> ?", "value"),
			Entry("contains", "contains", "title ILIKE ?", "%value%"),
			Entry("does not contains", "does not contains", "title NOT ILIKE ?", "%value%"),
			Entry("begins with", "begins with", "title ILIKE ?", "value%"),
			Entry("ends with", "ends with", "title ILIKE ?", "%value"),
		)
	})

	Describe("numberRule", func() {
		DescribeTable("operators",
			func(operator, expectedSql string, expectedValue int) {
				r := numberRule{Field: "year", Operator: operator, Value: 1985}
				sql, args, err := r.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(expectedSql))
				Expect(args).To(ConsistOf(expectedValue))
			},
			Entry("is", "is", "year = ?", 1985),
			Entry("is not", "is not", "year <> ?", 1985),
			Entry("is greater than", "is greater than", "year > ?", 1985),
			Entry("is less than", "is less than", "year < ?", 1985),
		)

		It("implements the 'is in the range' operator", func() {
			r := numberRule{Field: "year", Operator: "is in the range", Value: []int{1981, 1990}}
			sql, args, err := r.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(Equal("(year >= ? AND year <= ?)"))
			Expect(args).To(ConsistOf(1981, 1990))
		})
	})

	Describe("dateRule", func() {
		dateStr := "2021-10-14"
		date, _ := time.Parse("2006-01-02", dateStr)
		DescribeTable("simple operators",
			func(operator, expectedSql string, expectedValue time.Time) {
				r := dateRule{Field: "lastPlayed", Operator: operator, Value: dateStr}
				sql, args, err := r.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(expectedSql))
				Expect(args).To(ConsistOf(expectedValue))
			},
			Entry("is", "is", "lastPlayed = ?", date),
			Entry("is not", "is not", "lastPlayed <> ?", date),
			Entry("is before", "is before", "lastPlayed < ?", date),
			Entry("is after", "is after", "lastPlayed > ?", date),
		)

		DescribeTable("period operators",
			func(operator, expectedSql string, expectedValue time.Time) {
				delta := 30 * time.Hour // Must be large to account for the hours of the day
				r := dateRule{Field: "lastPlayed", Operator: operator, Value: 90}
				sql, args, err := r.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(expectedSql))
				Expect(args).To(ConsistOf(BeTemporally("~", expectedValue, delta)))
			},
			Entry("in the last", "in the last", "lastPlayed > ?", date.Add(-90*24*time.Hour)),
			Entry("not in the last", "not in the last", "lastPlayed < ?", date.Add(-90*24*time.Hour)),
		)

		It("accepts string as the 'in the last' operator value", func() {
			r := dateRule{Field: "lastPlayed", Operator: "in the last", Value: "90"}
			_, args, _ := r.ToSql()
			Expect(args).To(ConsistOf(BeTemporally("~", date.Add(-90*24*time.Hour), 30*time.Hour)))
		})

		It("implements the 'is in the range' operator", func() {
			date2Str := "2021-09-14"
			date2, _ := time.Parse("2006-01-02", date2Str)

			r := dateRule{Field: "lastPlayed", Operator: "is in the range", Value: []string{date2Str, dateStr}}
			sql, args, err := r.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(Equal("(lastPlayed >= ? AND lastPlayed <= ?)"))
			Expect(args).To(ConsistOf(BeTemporally("~", date2, 30*time.Hour), BeTemporally("~", date, 30*time.Hour)))
		})

		It("returns error if date is invalid", func() {
			r := dateRule{Field: "lastPlayed", Operator: "is", Value: "INVALID"}
			_, _, err := r.ToSql()
			Expect(err).To(MatchError("invalid date: INVALID"))
		})
	})

	Describe("boolRule", func() {
		DescribeTable("operators",
			func(operator, expectedSql string, expectedValue ...interface{}) {
				r := boolRule{Field: "loved", Operator: operator}
				sql, args, err := r.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(expectedSql))
				Expect(args).To(ConsistOf(expectedValue...))
			},
			Entry("is true", "is true", "loved = ?", true),
			Entry("is false", "is false", "loved = ?", false),
		)
	})
})
