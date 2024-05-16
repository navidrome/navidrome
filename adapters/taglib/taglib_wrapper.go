package taglib

/*
#cgo pkg-config: taglib
#cgo illumos LDFLAGS: -lstdc++ -lsendfile
#cgo linux darwin CXXFLAGS: -std=c++11
#cgo darwin LDFLAGS: -L/opt/homebrew/opt/taglib/lib
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "taglib_wrapper.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/navidrome/navidrome/log"
)

const iTunesKeyPrefix = "----:com.apple.itunes:"

func Version() string {
	return C.GoString(C.taglib_version())
}

func Read(filename string) (tags map[string][]string, err error) {
	// Do not crash on failures in the C code/library
	debug.SetPanicOnFault(true)
	defer func() {
		if r := recover(); r != nil {
			log.Error("extractor: recovered from panic when reading tags", "file", filename, "error", r)
			err = fmt.Errorf("extractor: recovered from panic: %s", r)
		}
	}()

	fp := getFilename(filename)
	defer C.free(unsafe.Pointer(fp))
	id, m := newMap()
	defer deleteMap(id)

	log.Trace("extractor: reading tags", "filename", filename, "map_id", id)
	res := C.taglib_read(fp, C.ulong(id))
	switch res {
	case C.TAGLIB_ERR_PARSE:
		// Check additional case whether the file is unreadable due to permission
		file, fileErr := os.OpenFile(filename, os.O_RDONLY, 0600)
		defer file.Close()

		if os.IsPermission(fileErr) {
			return nil, fmt.Errorf("navidrome does not have permission: %w", fileErr)
		} else if fileErr != nil {
			return nil, fmt.Errorf("cannot parse file media file: %w", fileErr)
		} else {
			return nil, fmt.Errorf("cannot parse file media file")
		}
	case C.TAGLIB_ERR_AUDIO_PROPS:
		return nil, fmt.Errorf("can't get audio properties from file")
	}
	if log.IsGreaterOrEqualTo(log.LevelDebug) {
		j, _ := json.Marshal(m)
		log.Trace("extractor: read tags", "tags", string(j), "filename", filename, "id", id)
	} else {
		log.Trace("extractor: read tags", "tags", m, "filename", filename, "id", id)
	}

	return m, nil
}

var lock sync.RWMutex
var allMaps = make(map[uint32]map[string][]string)
var mapsNextID uint32

func newMap() (id uint32, m map[string][]string) {
	lock.Lock()
	defer lock.Unlock()
	id = mapsNextID
	mapsNextID++
	m = make(map[string][]string)
	allMaps[id] = m
	return
}

func deleteMap(id uint32) {
	lock.Lock()
	defer lock.Unlock()
	delete(allMaps, id)
}

//export go_map_put_m4a_str
func go_map_put_m4a_str(id C.ulong, key *C.char, val *C.char) {
	k := strings.ToLower(C.GoString(key))

	// Special for M4A, do not catch keys that have no actual name
	k = strings.TrimPrefix(k, iTunesKeyPrefix)
	do_put_map(id, k, val)
}

//export go_map_put_str
func go_map_put_str(id C.ulong, key *C.char, val *C.char) {
	k := strings.ToLower(C.GoString(key))
	do_put_map(id, k, val)
}

//export go_map_put_lyrics
func go_map_put_lyrics(id C.ulong, lang *C.char, val *C.char) {
	k := "lyrics-" + strings.ToLower(C.GoString(lang))
	do_put_map(id, k, val)
}

func do_put_map(id C.ulong, key string, val *C.char) {
	if key == "" {
		return
	}

	lock.RLock()
	defer lock.RUnlock()
	m := allMaps[uint32(id)]
	v := strings.TrimSpace(C.GoString(val))
	m[key] = append(m[key], v)
}

/*
As I'm working on the new scanner, I see that the `properties` from extractor is ill-suited to extract multi-valued ID3 frames. I'll have to change the way we do it for ID3, probably by sending the raw frames to Go and mapping there, instead of relying on the auto-mapped `properties`.  I think this would reduce our reliance on C++, while also giving us more flexibility, including parsing the USLT / SYLT frames in Go
*/

//export go_map_put_int
func go_map_put_int(id C.ulong, key *C.char, val C.int) {
	valStr := strconv.Itoa(int(val))
	vp := C.CString(valStr)
	defer C.free(unsafe.Pointer(vp))
	go_map_put_str(id, key, vp)
}

//export go_map_put_lyric_line
func go_map_put_lyric_line(id C.ulong, lang *C.char, text *C.char, time C.int) {
	language := C.GoString(lang)
	line := C.GoString(text)
	timeGo := int64(time)

	ms := timeGo % 1000
	timeGo /= 1000
	sec := timeGo % 60
	timeGo /= 60
	min := timeGo % 60
	formatted_line := fmt.Sprintf("[%02d:%02d.%02d]%s\n", min, sec, ms/10, line)

	lock.RLock()
	defer lock.RUnlock()

	key := "lyrics-" + language

	m := allMaps[uint32(id)]
	existing, ok := m[key]
	if ok {
		existing[0] += formatted_line
	} else {
		m[key] = []string{formatted_line}
	}
}
