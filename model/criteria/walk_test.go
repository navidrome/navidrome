package criteria

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type unknownExpression struct{}

func (unknownExpression) criteriaExpression() {}

var _ = Describe("Walk", func() {
	It("visits the expression tree depth-first", func() {
		expr := All{
			Contains{"title": "love"},
			Any{
				Is{"album": "best of"},
				Gt{"rating": 3},
			},
		}

		var visited []string
		err := Walk(expr, func(expr Expression) error {
			visited = append(visited, fmt.Sprintf("%T", expr))
			return nil
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(visited).To(gomega.Equal([]string{
			"criteria.All",
			"criteria.Contains",
			"criteria.Any",
			"criteria.Is",
			"criteria.Gt",
		}))
	})

	It("stops when the visitor returns an error", func() {
		expectedErr := fmt.Errorf("stop")

		err := Walk(All{Contains{"title": "love"}}, func(Expression) error {
			return expectedErr
		})

		gomega.Expect(err).To(gomega.MatchError(expectedErr))
	})

	It("returns fields for leaf expressions", func() {
		gomega.Expect(Fields(Contains{"title": "love"})).To(gomega.Equal(map[string]any{"title": "love"}))
		gomega.Expect(Fields(After{"date": "2020-01-01"})).To(gomega.Equal(map[string]any{"date": "2020-01-01"}))
	})

	It("returns nil fields for group expressions", func() {
		gomega.Expect(Fields(All{Contains{"title": "love"}})).To(gomega.BeNil())
	})

	It("returns an error for unknown expression types", func() {
		err := Walk(unknownExpression{}, func(Expression) error { return nil })

		gomega.Expect(err).To(gomega.MatchError("unknown criteria expression type criteria.unknownExpression"))
	})
})
