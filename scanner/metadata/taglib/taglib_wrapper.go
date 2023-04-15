package taglib

/*
#cgo pkg-config: taglib
#cgo !illumos LDFLAGS: -lstdc++
#cgo illumos LDFLAGS: -lstdc++ -lsendfile
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
