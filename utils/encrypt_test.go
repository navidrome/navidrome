package utils

import (
	"context"
	"crypto/sha256"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("encrypt", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("decrypts correctly when using the same encryption key", func() {
		sum := sha256.Sum256([]byte("password"))
		encKey := sum[0:]
		data := "Can you keep a secret?"

		encrypted, err := Encrypt(ctx, encKey, data)
		Expect(err).ToNot(HaveOccurred())
		decrypted, err := Decrypt(ctx, encKey, encrypted)
		Expect(err).ToNot(HaveOccurred())

		Expect(decrypted).To(Equal(data))
	})

	It("fails to decrypt if not using the same encryption key", func() {
		sum := sha256.Sum256([]byte("password"))
		encKey := sum[0:]
		data := "Can you keep a secret?"

		encrypted, err := Encrypt(ctx, encKey, data)
		Expect(err).ToNot(HaveOccurred())

		sum = sha256.Sum256([]byte("different password"))
		encKey = sum[0:]
		_, err = Decrypt(ctx, encKey, encrypted)
		Expect(err).To(MatchError("cipher: message authentication failed"))
	})
})
