package model_test

import (
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Grant", func() {
	Describe("LastActivity", func() {
		created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

		It("is the creation time for a grant never used", func() {
			Expect(model.Grant{CreatedAt: created}.LastActivity()).To(Equal(created))
		})

		It("is the last use once the grant was used", func() {
			used := created.Add(time.Hour)
			Expect(model.Grant{CreatedAt: created, LastUsedAt: &used}.LastActivity()).To(Equal(used))
		})
	})
})
