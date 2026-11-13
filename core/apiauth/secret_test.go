package apiauth

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("grant secrets", func() {
	It("are ndg_ plus 22 base62 characters, hashed as hex SHA-256", func() {
		secret, hash := newSecret()
		Expect(secret).To(MatchRegexp(`^ndg_[0-9A-Za-z]{22}$`))
		Expect(hash).To(MatchRegexp(`^[0-9a-f]{64}$`))
		Expect(hashSecret(secret)).To(Equal(hash))
	})
	It("are unique", func() {
		a, _ := newSecret()
		b, _ := newSecret()
		Expect(a).ToNot(Equal(b))
		Expect(regexp.MustCompile(`^ndg_`).MatchString(a)).To(BeTrue())
	})
})
