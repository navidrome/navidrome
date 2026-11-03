package apiv1

import (
	"github.com/navidrome/navidrome/core/apiauth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("toScopes", func() {
	It("only produces scopes the spec's Scope enum allows", func() {
		for _, s := range toScopes(append([]string{apiauth.ScopeAll}, apiauth.KnownScopes...)) {
			Expect(s.Valid()).To(BeTrue(), "scope %q is missing from the spec's Scope enum", s)
		}
	})
})
