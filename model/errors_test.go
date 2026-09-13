package model_test

import (
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ErrNotFound", func() {
	It("is rest.ErrNotFound, so REST endpoints respond 404 instead of 500", func() {
		Expect(model.ErrNotFound).To(BeIdenticalTo(rest.ErrNotFound))
	})
})
