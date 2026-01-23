package storage

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage Test Suite")
}

var _ = Describe("Storage", func() {
	When("schema is not registered", func() {
		BeforeEach(func() {
			registry = map[string]constructor{}
		})

		It("should return error", func() {
			_, err := For("file:///tmp")
			Expect(err).To(HaveOccurred())
		})
	})
	When("schema is registered", func() {
		BeforeEach(func() {
			registry = map[string]constructor{}
			Register("file", func(url url.URL) Storage { return &fakeLocalStorage{u: url} })
			Register("s3", func(url url.URL) Storage { return &fakeS3Storage{u: url} })
		})

		It("should return correct implementation", func() {
			s, err := For("file:///tmp")
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(BeAssignableToTypeOf(&fakeLocalStorage{}))
			Expect(s.(*fakeLocalStorage).u.Scheme).To(Equal("file"))
			Expect(s.(*fakeLocalStorage).u.Path).To(Equal("/tmp"))

			s, err = For("s3:///bucket")
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(BeAssignableToTypeOf(&fakeS3Storage{}))
			Expect(s.(*fakeS3Storage).u.Scheme).To(Equal("s3"))
			Expect(s.(*fakeS3Storage).u.Path).To(Equal("/bucket"))
		})
		It("should return a file implementation when schema is not specified", func() {
			s, err := For("/tmp")
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(BeAssignableToTypeOf(&fakeLocalStorage{}))
			Expect(s.(*fakeLocalStorage).u.Scheme).To(Equal("file"))
			Expect(s.(*fakeLocalStorage).u.Path).To(Equal("/tmp"))
		})
		It("should return a file implementation for a relative folder", func() {
			s, err := For("tmp")
			Expect(err).ToNot(HaveOccurred())
			cwd, _ := os.Getwd()
			Expect(s).To(BeAssignableToTypeOf(&fakeLocalStorage{}))
			Expect(s.(*fakeLocalStorage).u.Scheme).To(Equal("file"))
			Expect(s.(*fakeLocalStorage).u.Path).To(Equal(filepath.Join(cwd, "tmp")))
		})
		It("should return error if schema is unregistered", func() {
			_, err := For("webdav:///tmp")
			Expect(err).To(HaveOccurred())
		})
		DescribeTable("should handle paths with special characters correctly",
			func(inputPath string) {
				s, err := For(inputPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(s).To(BeAssignableToTypeOf(&fakeLocalStorage{}))
				Expect(s.(*fakeLocalStorage).u.Scheme).To(Equal("file"))
				// The path should be exactly the same as the input - after URL parsing it gets decoded back
				Expect(s.(*fakeLocalStorage).u.Path).To(Equal(inputPath))
			},
			Entry("hash symbols", "/tmp/test#folder/file.mp3"),
			Entry("spaces", "/tmp/test folder/file with spaces.mp3"),
			Entry("question marks", "/tmp/test?query/file.mp3"),
			Entry("ampersands", "/tmp/test&amp/file.mp3"),
			Entry("multiple special chars", "/tmp/Song #1 & More?.mp3"),
		)
	})
})

type fakeLocalStorage struct {
	Storage
	u url.URL
}
type fakeS3Storage struct {
	Storage
	u url.URL
}
