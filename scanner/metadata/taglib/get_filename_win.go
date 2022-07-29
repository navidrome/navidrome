//go:build windows

package taglib

// From https://github.com/orofarne/gowchar

/*
#include <wchar.h>

const size_t SIZEOF_WCHAR_T = sizeof(wchar_t);

void gowchar_set (wchar_t *arr, int pos, wchar_t val)
{
	arr[pos] = val;
}

wchar_t gowchar_get (wchar_t *arr, int pos)
{
	return arr[pos];
}
*/
import "C"

import (
	"fmt"
	"unicode/utf16"
	"unicode/utf8"
)

var SIZEOF_WCHAR_T C.size_t = C.size_t(C.SIZEOF_WCHAR_T)

func getFilename(s string) *C.wchar_t {
	wstr, _ := StringToWcharT(s)
	return wstr
}

func StringToWcharT(s string) (*C.wchar_t, C.size_t) {
	switch SIZEOF_WCHAR_T {
	case 2:
		return stringToWchar2(s) // Windows
	case 4:
		return stringToWchar4(s) // Unix
	default:
		panic(fmt.Sprintf("Invalid sizeof(wchar_t) = %v", SIZEOF_WCHAR_T))
	}
	panic("?!!")
}

// Windows
func stringToWchar2(s string) (*C.wchar_t, C.size_t) {
	var slen int
	s1 := s
	for len(s1) > 0 {
		r, size := utf8.DecodeRuneInString(s1)
		if er, _ := utf16.EncodeRune(r); er == '\uFFFD' {
			slen += 1
		} else {
			slen += 2
		}
		s1 = s1[size:]
	}
	slen++ // \0
	res := C.malloc(C.size_t(slen) * SIZEOF_WCHAR_T)
	var i int
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r1, r2 := utf16.EncodeRune(r); r1 != '\uFFFD' {
			C.gowchar_set((*C.wchar_t)(res), C.int(i), C.wchar_t(r1))
			i++
			C.gowchar_set((*C.wchar_t)(res), C.int(i), C.wchar_t(r2))
			i++
		} else {
			C.gowchar_set((*C.wchar_t)(res), C.int(i), C.wchar_t(r))
			i++
		}
		s = s[size:]
	}
	C.gowchar_set((*C.wchar_t)(res), C.int(slen-1), C.wchar_t(0)) // \0
	return (*C.wchar_t)(res), C.size_t(slen)
}

// Unix
func stringToWchar4(s string) (*C.wchar_t, C.size_t) {
	slen := utf8.RuneCountInString(s)
	slen++ // \0
	res := C.malloc(C.size_t(slen) * SIZEOF_WCHAR_T)
	var i int
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		C.gowchar_set((*C.wchar_t)(res), C.int(i), C.wchar_t(r))
		s = s[size:]
		i++
	}
	C.gowchar_set((*C.wchar_t)(res), C.int(slen-1), C.wchar_t(0)) // \0
	return (*C.wchar_t)(res), C.size_t(slen)
}
