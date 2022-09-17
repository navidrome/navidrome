package taglib

/*
#cgo pkg-config: taglib
#cgo LDFLAGS: -lstdc++
#cgo linux darwin CXXFLAGS: -std=c++11
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "taglib_wrapper.h"
*/
import "C"
import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/navidrome/navidrome/log"
)

func Read(filename string) (tags map[string][]string, err error) {
	// Do not crash on failures in the C code/library
	debug.SetPanicOnFault(true)
	defer func() {
		if r := recover(); r != nil {
			log.Error("TagLib: recovered from panic when reading tags", "file", filename, "error", r)
			err = fmt.Errorf("TagLib: recovered from panic: %s", r)
		}
	}()

	fp := getFilename(filename)
	defer C.free(unsafe.Pointer(fp))
	id, m := newMap()
	defer deleteMap(id)

	res := C.taglib_read(fp, C.ulong(id))
	if log.CurrentLevel() >= log.LevelDebug {
		switch res {
		case C.TAGLIB_ERR_PARSE:
			// Check additional case whether the file is unreadable due to permission
			file, fileErr := os.OpenFile(filename, os.O_RDONLY, 0600)
			if fileErr != nil && os.IsPermission(fileErr) {
				log.Warn("Navidrome does not have permission to read media file", "filename", filename)
			} else {
				log.Warn("TagLib: cannot parse file", "filename", filename)
			}
			file.Close()
		case C.TAGLIB_ERR_AUDIO_PROPS:
			log.Warn("TagLib: can't get audio properties", "filename", filename)
		}
	}

	if res != 0 {
		return nil, fmt.Errorf("cannot process %s", filename)
	}
	log.Trace("TagLib: read tags", "tags", m, "filename", filename, "id", id)
	return m, nil
}

var lock sync.RWMutex
var maps = make(map[uint32]map[string][]string)
var mapsNextID uint32

func newMap() (id uint32, m map[string][]string) {
	lock.Lock()
	defer lock.Unlock()
	id = mapsNextID
	mapsNextID++
	m = make(map[string][]string)
	maps[id] = m
	return
}

func deleteMap(id uint32) {
	lock.Lock()
	defer lock.Unlock()
	delete(maps, id)
}

//export go_map_put_str
func go_map_put_str(id C.ulong, key *C.char, val *C.char) {
	lock.RLock()
	defer lock.RUnlock()
	m := maps[uint32(id)]
	k := strings.ToLower(C.GoString(key))
	v := strings.TrimSpace(C.GoString(val))
	m[k] = append(m[k], v)
}

//export go_map_put_int
func go_map_put_int(id C.ulong, key *C.char, val C.int) {
	valStr := strconv.Itoa(int(val))
	vp := C.CString(valStr)
	defer C.free(unsafe.Pointer(vp))
	go_map_put_str(id, key, vp)
}
