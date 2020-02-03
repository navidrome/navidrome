package scanner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChangeDetector", func() {
	var testFolder string
	var scanner *ChangeDetector

	lastModifiedSince := time.Time{}

	BeforeEach(func() {
		testFolder, _ = ioutil.TempDir("", "navidrome_tests")
		err := os.MkdirAll(testFolder, 0777)
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
		lastModifiedSince = nowWithDelay()
		err = os.MkdirAll(filepath.Join(testFolder, "a"), 0777)
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf(".", P("a")))

		// Add more subfolders
		lastModifiedSince = nowWithDelay()
		err = os.MkdirAll(filepath.Join(testFolder, "a", "b", "c"), 0777)
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf(P("a"), P("a/b"), P("a/b/c")))

		// Scan with no changes
		lastModifiedSince = nowWithDelay()
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(BeEmpty())

		// New file in subfolder
		lastModifiedSince = nowWithDelay()
		_, err = os.Create(filepath.Join(testFolder, "a", "b", "empty.txt"))
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf(P("a/b")))

		// Delete file in subfolder
		lastModifiedSince = nowWithDelay()
		err = os.Remove(filepath.Join(testFolder, "a", "b", "empty.txt"))
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf(P("a/b")))

		// Delete subfolder
		lastModifiedSince = nowWithDelay()
		err = os.Remove(filepath.Join(testFolder, "a", "b", "c"))
		if err != nil {
			panic(err)
		}
		changed, deleted, err = scanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(ConsistOf(P("a/b/c")))
		Expect(changed).To(ConsistOf(P("a/b")))

		// Only returns changes after lastModifiedSince
		lastModifiedSince = nowWithDelay()
		newScanner := NewChangeDetector(testFolder)
		changed, deleted, err = newScanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(BeEmpty())
		Expect(changed).To(BeEmpty())

		f, err := os.Create(filepath.Join(testFolder, "a", "b", "new.txt"))
		f.Close()
		changed, deleted, err = newScanner.Scan(lastModifiedSince)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeEmpty())
		Expect(changed).To(ConsistOf(P("a/b")))
	})
})

// I hate time-based tests....
func nowWithDelay() time.Time {
	now := time.Now()
	time.Sleep(50 * time.Millisecond)
	return now
}
