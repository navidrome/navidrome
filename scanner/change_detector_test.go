package scanner

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChangeDetector", func() {
	var testFolder string
	var scanner *ChangeDetector

	lastModifiedSince := time.Time{}

	BeforeEach(func() {
		testFolder, _ = ioutil.TempDir("", "cloudsonic_tests")
		err := os.MkdirAll(testFolder, 0700)
		if err != nil {
			panic(err)
		}
		scanner = NewChangeDetector(testFolder)
	})

	It("detects changes recursively", func() {
		// Scan empty folder
		changed, deleted, err := scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf("."))

		// Add one subfolder
		err = os.MkdirAll(path.Join(testFolder, "a"), 0700)
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf(".", "/a"))

		// Add more subfolders
		err = os.MkdirAll(path.Join(testFolder, "a", "b", "c"), 0700)
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf("/a", "/a/b", "/a/b/c"))

		// Scan with no changes
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(BeEmpty())

		// New file in subfolder
		_, err = os.Create(path.Join(testFolder, "a", "b", "empty.txt"))
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf("/a/b"))

		// Delete file in subfolder
		err = os.Remove(path.Join(testFolder, "a", "b", "empty.txt"))
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf("/a/b"))

		// Delete subfolder
		err = os.Remove(path.Join(testFolder, "a", "b", "c"))
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(ConsistOf("/a/b/c"))
		Expect(changed).To(ConsistOf("/a/b"))

		// Only returns changes after lastModifiedSince
		lastModifiedSince = time.Now()
		newScanner := NewChangeDetector(testFolder)
		changed, deleted, err = newScanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(BeEmpty())
		Expect(changed).To(BeEmpty())

		_, err = os.Create(path.Join(testFolder, "a", "b", "new.txt"))
		changed, deleted, err = newScanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf("/a/b"))
	})
})
