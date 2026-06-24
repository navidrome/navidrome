package cache

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Spread FS", func() {
	var fs *spreadFS
	var rootDir string

	BeforeEach(func() {
		var err error
		rootDir, _ = os.MkdirTemp("", "spread_fs")
		fs, err = NewSpreadFS(rootDir, 0755)
		Expect(err).To(BeNil())
	})
	AfterEach(func() {
		_ = os.RemoveAll(rootDir)
	})

	Describe("KeyMapper", func() {
		It("creates a file with proper name format", func() {
			mapped := fs.KeyMapper("abc")
			Expect(mapped).To(HavePrefix(fs.root))
			mapped = strings.TrimPrefix(mapped, fs.root)
			parts := strings.Split(mapped, string(filepath.Separator))
			Expect(parts).To(HaveLen(4))
			Expect(parts[3]).To(HaveLen(40))
		})
		It("returns the unmodified key if it is a cache file path", func() {
			mapped := fs.KeyMapper("abc")
			Expect(mapped).To(HavePrefix(fs.root))
			Expect(fs.KeyMapper(mapped)).To(Equal(mapped))
		})
	})

	Describe("MarkComplete / Remove markers", func() {
		It("creates a .complete marker for a data file", func() {
			data := fs.KeyMapper("song1")
			f, err := fs.Create(data)
			Expect(err).To(BeNil())
			_, _ = f.Write([]byte("ok"))
			_ = f.Close()

			Expect(fs.MarkComplete(data)).To(Succeed())
			_, statErr := os.Stat(data + ".complete")
			Expect(statErr).To(BeNil())
		})

		It("removes the sibling marker when the data file is removed", func() {
			data := fs.KeyMapper("song2")
			f, err := fs.Create(data)
			Expect(err).To(BeNil())
			_, _ = f.Write([]byte("ok"))
			_ = f.Close()
			Expect(fs.MarkComplete(data)).To(Succeed())

			Expect(fs.Remove(data)).To(Succeed())
			_, dataErr := os.Stat(data)
			Expect(os.IsNotExist(dataErr)).To(BeTrue())
			_, markErr := os.Stat(data + ".complete")
			Expect(os.IsNotExist(markErr)).To(BeTrue())
		})
	})

	Describe("Reload", func() {
		makeData := func(content string) string {
			file := fs.KeyMapper(content)
			f, err := fs.Create(file)
			Expect(err).To(BeNil())
			_, _ = f.Write([]byte(content))
			_ = f.Close()
			return file
		}

		It("migrates all existing files on first run and writes the sentinel", func() {
			for _, c := range []string{"aaaaa", "bbbbb", "ccccc"} {
				makeData(c) // no markers, simulating a pre-upgrade cache
			}

			var actual []string
			err := fs.Reload(func(key, name string) {
				Expect(key).To(Equal(name))
				data, _ := os.ReadFile(name)
				actual = append(actual, string(data))
			})
			Expect(err).To(BeNil())
			Expect(actual).To(ContainElements("aaaaa", "bbbbb", "ccccc"))
			Expect(actual).To(HaveLen(3))

			_, sentinelErr := os.Stat(filepath.Join(rootDir, ".nd-migrated"))
			Expect(sentinelErr).To(BeNil())
		})

		It("after migration, adopts only marked files and deletes unmarked partials", func() {
			// Pretend migration already happened.
			Expect(os.WriteFile(filepath.Join(rootDir, ".nd-migrated"), nil, 0600)).To(Succeed())

			good := makeData("good")
			Expect(fs.MarkComplete(good)).To(Succeed())
			bad := makeData("bad") // partial: no marker

			var actual []string
			err := fs.Reload(func(key, name string) { actual = append(actual, name) })
			Expect(err).To(BeNil())
			Expect(actual).To(ConsistOf(good))

			_, badErr := os.Stat(bad)
			Expect(os.IsNotExist(badErr)).To(BeTrue()) // partial deleted
		})

		It("ignores and cleans orphan markers", func() {
			Expect(os.WriteFile(filepath.Join(rootDir, ".nd-migrated"), nil, 0600)).To(Succeed())
			orphan := fs.KeyMapper("orphan") + ".complete"
			Expect(os.MkdirAll(filepath.Dir(orphan), 0755)).To(Succeed())
			Expect(os.WriteFile(orphan, nil, 0600)).To(Succeed())

			var actual []string
			err := fs.Reload(func(key, name string) { actual = append(actual, name) })
			Expect(err).To(BeNil())
			Expect(actual).To(BeEmpty())
			_, orphanErr := os.Stat(orphan)
			Expect(os.IsNotExist(orphanErr)).To(BeTrue())
		})
	})
})
