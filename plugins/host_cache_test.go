package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/plugins/host/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CacheService", func() {
	var service *cacheServiceImpl
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		service = newCacheService("test_plugin")
	})

	Describe("getTTL", func() {
		It("returns default TTL when seconds is 0", func() {
			ttl := service.getTTL(0)
			Expect(ttl).To(Equal(defaultCacheTTL))
		})

		It("returns default TTL when seconds is negative", func() {
			ttl := service.getTTL(-10)
			Expect(ttl).To(Equal(defaultCacheTTL))
		})

		It("returns correct duration when seconds is positive", func() {
			ttl := service.getTTL(60)
			Expect(ttl).To(Equal(time.Minute))
		})
	})

	Describe("String Operations", func() {
		It("sets and gets a string value", func() {
			_, err := service.SetString(ctx, &cache.SetStringRequest{
				Key:        "string_key",
				Value:      "test_value",
				TtlSeconds: 300,
			})
			Expect(err).NotTo(HaveOccurred())

			res, err := service.GetString(ctx, &cache.GetRequest{Key: "string_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeTrue())
			Expect(res.Value).To(Equal("test_value"))
		})

		It("returns not exists for missing key", func() {
			res, err := service.GetString(ctx, &cache.GetRequest{Key: "missing_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeFalse())
		})
	})

	Describe("Integer Operations", func() {
		It("sets and gets an integer value", func() {
			_, err := service.SetInt(ctx, &cache.SetIntRequest{
				Key:        "int_key",
				Value:      42,
				TtlSeconds: 300,
			})
			Expect(err).NotTo(HaveOccurred())

			res, err := service.GetInt(ctx, &cache.GetRequest{Key: "int_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeTrue())
			Expect(res.Value).To(Equal(int64(42)))
		})
	})

	Describe("Float Operations", func() {
		It("sets and gets a float value", func() {
			_, err := service.SetFloat(ctx, &cache.SetFloatRequest{
				Key:        "float_key",
				Value:      3.14,
				TtlSeconds: 300,
			})
			Expect(err).NotTo(HaveOccurred())

			res, err := service.GetFloat(ctx, &cache.GetRequest{Key: "float_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeTrue())
			Expect(res.Value).To(Equal(3.14))
		})
	})

	Describe("Bytes Operations", func() {
		It("sets and gets a bytes value", func() {
			byteData := []byte("hello world")
			_, err := service.SetBytes(ctx, &cache.SetBytesRequest{
				Key:        "bytes_key",
				Value:      byteData,
				TtlSeconds: 300,
			})
			Expect(err).NotTo(HaveOccurred())

			res, err := service.GetBytes(ctx, &cache.GetRequest{Key: "bytes_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeTrue())
			Expect(res.Value).To(Equal(byteData))
		})
	})

	Describe("Type mismatch handling", func() {
		It("returns not exists when type doesn't match the getter", func() {
			// Set string
			_, err := service.SetString(ctx, &cache.SetStringRequest{
				Key:   "mixed_key",
				Value: "string value",
			})
			Expect(err).NotTo(HaveOccurred())

			// Try to get as int
			res, err := service.GetInt(ctx, &cache.GetRequest{Key: "mixed_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeFalse())
		})
	})

	Describe("Remove Operation", func() {
		It("removes a value from the cache", func() {
			// Set a value
			_, err := service.SetString(ctx, &cache.SetStringRequest{
				Key:   "remove_key",
				Value: "to be removed",
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify it exists
			res, err := service.Has(ctx, &cache.HasRequest{Key: "remove_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeTrue())

			// Remove it
			_, err = service.Remove(ctx, &cache.RemoveRequest{Key: "remove_key"})
			Expect(err).NotTo(HaveOccurred())

			// Verify it's gone
			res, err = service.Has(ctx, &cache.HasRequest{Key: "remove_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeFalse())
		})
	})

	Describe("Has Operation", func() {
		It("returns true for existing key", func() {
			// Set a value
			_, err := service.SetString(ctx, &cache.SetStringRequest{
				Key:   "existing_key",
				Value: "exists",
			})
			Expect(err).NotTo(HaveOccurred())

			// Check if it exists
			res, err := service.Has(ctx, &cache.HasRequest{Key: "existing_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeTrue())
		})

		It("returns false for non-existing key", func() {
			res, err := service.Has(ctx, &cache.HasRequest{Key: "non_existing_key"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Exists).To(BeFalse())
		})
	})
})
