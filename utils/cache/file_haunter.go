package cache

import (
	"sort"
	"time"

	"github.com/djherbis/fscache"
	"github.com/dustin/go-humanize"
	"github.com/navidrome/navidrome/log"
)

type haunterKV struct {
	key   string
	value fscache.Entry
	info  fscache.FileInfo
}

// NewFileHaunter returns a simple haunter which runs every "period"
// and scrubs older files when the total file size is over maxSize or
// total item count is over maxItems.  It also removes empty (invalid) files.
// If maxItems or maxSize are 0, they won't be checked
//
// Based on fscache.NewLRUHaunter
func NewFileHaunter(name string, maxItems int, maxSize uint64, period time.Duration) fscache.LRUHaunter {
	return &fileHaunter{
		name:     name,
		period:   period,
		maxItems: maxItems,
		maxSize:  maxSize,
	}
}

type fileHaunter struct {
	name     string
	period   time.Duration
	maxItems int
	maxSize  uint64
}

func (j *fileHaunter) Next() time.Duration {
	return j.period
}

func (j *fileHaunter) Scrub(c fscache.CacheAccessor) (keysToReap []string) {
	var count int
	var size uint64
	var okFiles []haunterKV

	log.Trace("Running cache cleanup", "cache", j.name, "maxSize", humanize.Bytes(j.maxSize))
	c.EnumerateEntries(func(key string, e fscache.Entry) bool {
		if e.InUse() {
			return true
		}

		fileInfo, err := c.Stat(e.Name())
		if err != nil {
			return true
		}

		if fileInfo.Size() == 0 {
			log.Trace("Removing invalid empty file", "file", e.Name())
			keysToReap = append(keysToReap, key)
		}

		count++
		size = size + uint64(fileInfo.Size())
		okFiles = append(okFiles, haunterKV{
			key:   key,
			value: e,
			info:  fileInfo,
		})

		return true
	})

	sort.Slice(okFiles, func(i, j int) bool {
		iLastRead := okFiles[i].info.AccessTime()
		jLastRead := okFiles[j].info.AccessTime()

		return iLastRead.Before(jLastRead)
	})

	collectKeysToReapFn := func() bool {
		var key *string
		var err error
		key, count, size, err = j.removeFirst(&okFiles, count, size)
		if err != nil {
			return false
		}
		if key != nil {
			keysToReap = append(keysToReap, *key)
		}

		return true
	}

	log.Trace("Current cache stats", "cache", j.name, "size", humanize.Bytes(size), "numItems", count)

	if j.maxItems > 0 {
		for count > j.maxItems {
			if !collectKeysToReapFn() {
				break
			}
		}
	}

	if j.maxSize > 0 {
		for size > j.maxSize {
			if !collectKeysToReapFn() {
				break
			}
		}
	}

	if len(keysToReap) > 0 {
		log.Trace("Removing items from cache", "cache", j.name, "numItems", len(keysToReap))
	}
	return keysToReap
}

func (j *fileHaunter) removeFirst(items *[]haunterKV, count int, size uint64) (*string, int, uint64, error) {
	var f haunterKV

	f, *items = (*items)[0], (*items)[1:]

	count--
	size = size - uint64(f.info.Size())

	return &f.key, count, size, nil
}
