package conf_test

import (
	"os"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dir", func() {
	Describe("NewDir", func() {
		It("creates a Dir with the given path without side effects", func() {
			d := conf.NewDir("/some/path")
			Expect(d.String()).To(Equal("/some/path"))
		})
	})

	Describe("String", func() {
		It("returns the raw path without creating the directory", func() {
			d := conf.NewDir("/nonexistent/path/that/should/not/be/created")
			Expect(d.String()).To(Equal("/nonexistent/path/that/should/not/be/created"))
		})
	})

	Describe("Path", func() {
		It("creates the directory and returns the path on first call", func() {
			dir := GinkgoT().TempDir()
			target := dir + "/subdir/nested"
			d := conf.NewDir(target)

			path, err := d.Path()
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(target))
			Expect(target).To(BeADirectory())
		})

		It("returns the same result on subsequent calls (sync.Once)", func() {
			dir := GinkgoT().TempDir()
			target := dir + "/once"
			d := conf.NewDir(target)

			path1, err1 := d.Path()
			path2, err2 := d.Path()
			Expect(err1).ToNot(HaveOccurred())
			Expect(err2).ToNot(HaveOccurred())
			Expect(path1).To(Equal(path2))
		})

		It("returns an error when directory cannot be created", func() {
			f := GinkgoT().TempDir()
			blocker := f + "/blocker"
			By("creating a file that blocks directory creation")
			Expect(os.WriteFile(blocker, []byte("x"), 0600)).To(Succeed())
			invalid := blocker + "/subdir"

			d := conf.NewDir(invalid)
			_, pathErr := d.Path()
			Expect(pathErr).To(HaveOccurred())
		})

		It("returns empty path and no error for empty path", func() {
			d := conf.NewDir("")
			path, err := d.Path()
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(BeEmpty())
		})
	})

	Describe("MustPath", func() {
		It("returns the path when directory is created successfully", func() {
			dir := GinkgoT().TempDir()
			target := dir + "/mustpath"
			d := conf.NewDir(target)

			path := d.MustPath()
			Expect(path).To(Equal(target))
			Expect(target).To(BeADirectory())
		})

		It("calls logFatal on error", func() {
			var fatalMsg []any
			restore := conf.SetLogFatal(func(args ...any) {
				fatalMsg = args
				panic("logFatal called")
			})
			DeferCleanup(restore)

			f := GinkgoT().TempDir() + "/blocker"
			Expect(os.WriteFile(f, []byte("x"), 0600)).To(Succeed())
			invalid := f + "/subdir"

			d := conf.NewDir(invalid)
			Expect(func() { d.MustPath() }).To(Panic())
			Expect(fatalMsg).ToNot(BeEmpty())
		})
	})

	Describe("MarshalText", func() {
		It("returns the raw path bytes without side effects", func() {
			d := conf.NewDir("/marshal/path")
			b, err := d.MarshalText()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(b)).To(Equal("/marshal/path"))
		})
	})

	Describe("UnmarshalText", func() {
		It("sets the path from bytes without side effects", func() {
			d := conf.NewDir("")
			err := d.UnmarshalText([]byte("/unmarshal/path"))
			Expect(err).ToNot(HaveOccurred())
			Expect(d.String()).To(Equal("/unmarshal/path"))
		})

		It("allows round-trip marshal/unmarshal", func() {
			d1 := conf.NewDir("/round/trip")
			b, err := d1.MarshalText()
			Expect(err).ToNot(HaveOccurred())

			var d2 conf.Dir
			err = d2.UnmarshalText(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(d2.String()).To(Equal(d1.String()))
		})
	})
})
