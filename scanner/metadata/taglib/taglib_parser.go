//+build cgo

package taglib

/*
#cgo pkg-config: taglib
#cgo LDFLAGS: -lstdc++
#cgo linux CXXFLAGS: -std=c++11
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "taglib_parser.h"
*/
import "C"
import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/navidrome/navidrome/log"
)

func Read(filename string) (map[string]string, error) {
	fp := C.CString(filename)
	defer C.free(unsafe.Pointer(fp))
	id, m := newMap()
	defer deleteMap(id)

	res := C.taglib_read(fp, C.ulong(id))
	if log.CurrentLevel() >= log.LevelDebug {
		switch res {
		case C.TAGLIB_ERR_PARSE:
			log.Warn("TagLib: cannot parse file", "filename", filename)
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
var maps = make(map[uint32]map[string]string)
var mapsNextID uint32

func newMap() (id uint32, m map[string]string) {
	lock.Lock()
	defer lock.Unlock()
	id = mapsNextID
	mapsNextID++
	m = make(map[string]string)
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
	if _, ok := m[k]; !ok {
		v := strings.TrimSpace(C.GoString(val))
		m[k] = v
	}
}

//export go_map_put_int
func go_map_put_int(id C.ulong, key *C.char, val C.int) {
	valStr := strconv.Itoa(int(val))
	vp := C.CString(valStr)
	defer C.free(unsafe.Pointer(vp))
	go_map_put_str(id, key, vp)
}
