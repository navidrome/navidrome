package icy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReadHTTPStreamTitles", func() {
	const metaInt = 5

	It("requests ICY metadata and parses stream titles", func() {
		var icyHeader string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			icyHeader = r.Header.Get("Icy-MetaData")
			w.Header().Set("icy-metaint", "5")
			_, _ = w.Write(icyStream(metaInt, "StreamTitle='Artist - Title';"))
		}))
		defer server.Close()

		var titles []string
		err := ReadHTTPStreamTitles(context.Background(), server.Client(), server.URL, func(title string) {
			titles = append(titles, title)
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(icyHeader).To(Equal("1"))
		Expect(titles).To(Equal([]string{"Artist - Title"}))
	})

	It("follows redirects before reading stream metadata", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(w, r, "/stream", http.StatusFound)
				return
			}

			Expect(r.URL.Path).To(Equal("/stream"))
			Expect(r.Header.Get("Icy-MetaData")).To(Equal("1"))
			w.Header().Set("Icy-MetaInt", "5")
			_, _ = w.Write(icyStream(metaInt, "StreamTitle='Redirected - Title';"))
		}))
		defer server.Close()

		var titles []string
		err := ReadHTTPStreamTitles(context.Background(), server.Client(), server.URL+"/redirect", func(title string) {
			titles = append(titles, title)
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(titles).To(Equal([]string{"Redirected - Title"}))
	})

	It("returns an error when icy-metaint is missing", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not an icy stream"))
		}))
		defer server.Close()

		err := ReadHTTPStreamTitles(context.Background(), server.Client(), server.URL, func(string) {})

		Expect(errors.Is(err, ErrMissingMetaInt)).To(BeTrue())
	})

	It("returns an error when icy-metaint is invalid", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("icy-metaint", "zero")
			_, _ = w.Write([]byte("not an icy stream"))
		}))
		defer server.Close()

		err := ReadHTTPStreamTitles(context.Background(), server.Client(), server.URL, func(string) {})

		Expect(errors.Is(err, ErrInvalidMetaInt)).To(BeTrue())
	})

	It("returns response status errors for non-2xx responses", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "offline", http.StatusServiceUnavailable)
		}))
		defer server.Close()

		err := ReadHTTPStreamTitles(context.Background(), server.Client(), server.URL, func(string) {})

		var statusErr *ResponseStatusError
		Expect(errors.As(err, &statusErr)).To(BeTrue())
		Expect(statusErr.StatusCode).To(Equal(http.StatusServiceUnavailable))
	})

	It("stops when context is cancelled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := ReadHTTPStreamTitles(ctx, http.DefaultClient, "https://example.test/stream", func(string) {})

		Expect(errors.Is(err, context.Canceled)).To(BeTrue())
	})

	It("stops a body read when context is cancelled", func() {
		started := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("icy-metaint", "5")
			w.(http.Flusher).Flush()
			close(started)
			<-r.Context().Done()
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		errC := make(chan error, 1)
		go func() {
			errC <- ReadHTTPStreamTitles(ctx, server.Client(), server.URL, func(string) {})
		}()

		Eventually(started).Should(BeClosed())
		cancel()

		Eventually(errC).Should(Receive(MatchError(ContainSubstring("context canceled"))))
	})
})
