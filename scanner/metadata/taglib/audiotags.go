/***************************************************************************
   copyright            : (C) 2014 by Nick Sellen
   email                : code@nicksellen.co.uk
***************************************************************************/

/***************************************************************************
 *   This library is free software; you can redistribute it and/or modify  *
 *   it  under the terms of the GNU Lesser General Public License version  *
 *   2.1 as published by the Free Software Foundation.                     *
 *                                                                         *
 *   This library is distributed in the hope that it will be useful, but   *
 *   WITHOUT ANY WARRANTY; without even the implied warranty of            *
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU     *
 *   Lesser General Public License for more details.                       *
 *                                                                         *
 *   You should have received a copy of the GNU Lesser General Public      *
 *   License along with this library; if not, write to the Free Software   *
 *   Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  *
 *   USA                                                                   *
 ***************************************************************************/

package taglib

/*
#cgo pkg-config: taglib
#cgo LDFLAGS: -lstdc++
#include "audiotags.h"
#include <stdlib.h>
*/
import "C"
import (
	"strings"
	"sync"
	"unsafe"
)

import "fmt"

type File C.TagLib_File

type AudioProperties struct {
	Length, Bitrate, Samplerate, Channels int
}

func Open(filename string) (*File, error) {
	fp := C.CString(filename)
	defer C.free(unsafe.Pointer(fp))
	f := (C.audiotags_file_new(fp))
	if f == nil {
		return nil, fmt.Errorf("cannot process %s", filename)
	}
	return (*File)(f), nil
}

func Read(filename string) (map[string]string, *AudioProperties, error) {
	f, err := Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	return f.ReadTags(), f.ReadAudioProperties(), nil
}

func (f *File) Close() {
	C.audiotags_file_close((*C.TagLib_File)(f))
}

func (f *File) ReadTags() map[string]string {
	id, m := newMap()
	defer deleteMap(id)
	C.audiotags_file_properties((*C.TagLib_File)(f), C.ulong(id))
	return m
}

func (f *File) ReadAudioProperties() *AudioProperties {
	ap := C.audiotags_file_audioproperties((*C.TagLib_File)(f))
	if ap == nil {
		return nil
	}
	p := AudioProperties{}
	p.Length = int(C.audiotags_audioproperties_length(ap))
	p.Bitrate = int(C.audiotags_audioproperties_bitrate(ap))
	p.Samplerate = int(C.audiotags_audioproperties_samplerate(ap))
	p.Channels = int(C.audiotags_audioproperties_channels(ap))
	return &p
}

var lock sync.RWMutex
var maps = make(map[uint32]map[string]string)
var mapsNextId uint32

func newMap() (id uint32, m map[string]string) {
	lock.Lock()
	defer lock.Unlock()
	id = mapsNextId
	mapsNextId++
	m = make(map[string]string)
	maps[id] = m
	return
}

func deleteMap(id uint32) {
	lock.Lock()
	defer lock.Unlock()
	delete(maps, id)
}

//export go_map_put
func go_map_put(id C.ulong, key *C.char, val *C.char) {
	lock.RLock()
	defer lock.RUnlock()
	m := maps[uint32(id)]
	k := strings.ToLower(C.GoString(key))
	v := C.GoString(val)
	m[k] = v
}
