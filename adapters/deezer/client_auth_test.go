package deezer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JWT Authentication", func() {
	var httpClient *fakeHttpClient
	var client *client
	var ctx context.Context

	BeforeEach(func() {
		httpClient = &fakeHttpClient{}
		client = newClient(httpClient, "en")
		ctx = context.Background()
	})

	Describe("getJWT", func() {
		Context("with a valid JWT response", func() {
			It("successfully fetches and caches a JWT token", func() {
				testJWT := createTestJWT(5 * time.Minute)
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, testJWT))),
				})

				token, err := client.getJWT(ctx)
				Expect(err).To(BeNil())
				Expect(token).To(Equal(testJWT))
			})

			It("returns the cached token on subsequent calls", func() {
				testJWT := createTestJWT(5 * time.Minute)
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, testJWT))),
				})

				// First call should fetch from API
				token1, err := client.getJWT(ctx)
				Expect(err).To(BeNil())
				Expect(token1).To(Equal(testJWT))
				Expect(httpClient.lastRequest.URL.Path).To(Equal("/login/anonymous"))

				// Second call should return cached token without hitting API
				httpClient.lastRequest = nil // Clear last request to verify no new request is made
				token2, err := client.getJWT(ctx)
				Expect(err).To(BeNil())
				Expect(token2).To(Equal(testJWT))
				Expect(httpClient.lastRequest).To(BeNil()) // No new request made
			})

			It("parses the JWT expiration time correctly", func() {
				expectedExpiration := time.Now().Add(5 * time.Minute)
				testToken, err := jwt.NewBuilder().
					Expiration(expectedExpiration).
					Build()
				Expect(err).To(BeNil())
				testJWT, err := jwt.Sign(testToken, jwt.WithInsecureNoSignature())
				Expect(err).To(BeNil())

				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, string(testJWT)))),
				})

				token, err := client.getJWT(ctx)
				Expect(err).To(BeNil())
				Expect(token).ToNot(BeEmpty())

				// Verify the token is cached until close to expiration
				// The cache should expire 1 minute before the JWT expires
				expectedCacheExpiry := expectedExpiration.Add(-1 * time.Minute)
				Expect(client.jwt.expiresAt).To(BeTemporally("~", expectedCacheExpiry, 2*time.Second))
			})
		})

		Context("with JWT tokens that expire soon", func() {
			It("rejects tokens that expire in less than 1 minute", func() {
				// Create a token that expires in 30 seconds (less than 1-minute buffer)
				testJWT := createTestJWT(30 * time.Second)
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, testJWT))),
				})

				_, err := client.getJWT(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("JWT token already expired or expires too soon"))
			})

			It("rejects already expired tokens", func() {
				// Create a token that expired 1 minute ago
				testJWT := createTestJWT(-1 * time.Minute)
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, testJWT))),
				})

				_, err := client.getJWT(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("JWT token already expired or expires too soon"))
			})

			It("accepts tokens that expire in more than 1 minute", func() {
				// Create a token that expires in 2 minutes (just over the 1-minute buffer)
				testJWT := createTestJWT(2 * time.Minute)
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, testJWT))),
				})

				token, err := client.getJWT(ctx)
				Expect(err).To(BeNil())
				Expect(token).ToNot(BeEmpty())
			})
		})

		Context("with invalid responses", func() {
			It("handles HTTP error responses", func() {
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error":"Internal server error"}`)),
				})

				_, err := client.getJWT(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to get JWT token"))
			})

			It("handles malformed JSON responses", func() {
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{invalid json}`)),
				})

				_, err := client.getJWT(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse auth response"))
			})

			It("handles responses with empty JWT field", func() {
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{"jwt":""}`)),
				})

				_, err := client.getJWT(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("deezer: no JWT token in response"))
			})

			It("handles invalid JWT tokens", func() {
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{"jwt":"not-a-valid-jwt"}`)),
				})

				_, err := client.getJWT(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse JWT token"))
			})

			It("rejects JWT tokens without expiration", func() {
				// Create a JWT without expiration claim
				testToken, err := jwt.NewBuilder().
					Claim("custom", "value").
					Build()
				Expect(err).To(BeNil())

				// Verify token has no expiration
				Expect(testToken.Expiration().IsZero()).To(BeTrue())

				testJWT, err := jwt.Sign(testToken, jwt.WithInsecureNoSignature())
				Expect(err).To(BeNil())

				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, string(testJWT)))),
				})

				_, err = client.getJWT(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("deezer: JWT token has no expiration time"))
			})
		})

		Context("token caching behavior", func() {
			It("fetches a new token when the cached token expires", func() {
				// First token expires in 5 minutes
				firstJWT := createTestJWT(5 * time.Minute)
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, firstJWT))),
				})

				token1, err := client.getJWT(ctx)
				Expect(err).To(BeNil())
				Expect(token1).To(Equal(firstJWT))

				// Manually expire the cached token
				client.jwt.expiresAt = time.Now().Add(-1 * time.Second)

				// Second token with different expiration (10 minutes)
				secondJWT := createTestJWT(10 * time.Minute)
				httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s"}`, secondJWT))),
				})

				token2, err := client.getJWT(ctx)
				Expect(err).To(BeNil())
				Expect(token2).To(Equal(secondJWT))
				Expect(token2).ToNot(Equal(token1))
			})
		})
	})

	Describe("jwtToken cache", func() {
		var cache *jwtToken

		BeforeEach(func() {
			cache = &jwtToken{}
		})

		It("returns false for expired tokens", func() {
			cache.set("test-token", -1*time.Second) // Already expired
			token, valid := cache.get()
			Expect(valid).To(BeFalse())
			Expect(token).To(BeEmpty())
		})

		It("returns true for valid tokens", func() {
			cache.set("test-token", 4*time.Minute)
			token, valid := cache.get()
			Expect(valid).To(BeTrue())
			Expect(token).To(Equal("test-token"))
		})

		It("is thread-safe for concurrent access", func() {
			wg := sync.WaitGroup{}

			// Writer goroutine
			wg.Go(func() {
				for i := 0; i < 100; i++ {
					cache.set(fmt.Sprintf("token-%d", i), 1*time.Hour)
					time.Sleep(1 * time.Millisecond)
				}
			})

			// Reader goroutine
			wg.Go(func() {
				for i := 0; i < 100; i++ {
					cache.get()
					time.Sleep(1 * time.Millisecond)
				}
			})

			// Wait for both goroutines to complete
			wg.Wait()

			// Verify final state is valid
			token, valid := cache.get()
			Expect(valid).To(BeTrue())
			Expect(token).To(HavePrefix("token-"))
		})
	})
})

// createTestJWT creates a valid JWT token for testing purposes
func createTestJWT(expiresIn time.Duration) string {
	token, err := jwt.NewBuilder().
		Expiration(time.Now().Add(expiresIn)).
		Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create test JWT: %v", err))
	}
	signed, err := jwt.Sign(token, jwt.WithInsecureNoSignature())
	if err != nil {
		panic(fmt.Sprintf("failed to sign test JWT: %v", err))
	}
	return string(signed)
}
