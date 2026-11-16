package conf

import (
	"reflect"
	"sync/atomic"
)

var current atomic.Pointer[Options]

func init() {
	current.Store(&Options{})
}

// Server returns a copy of the current configuration. A published Options is never changed in place,
// so readers need no locking. Use Update to change it.
func Server() Options {
	return *current.Load()
}

// Update applies fn to a deep copy of the current configuration and publishes the result.
// fn may run more than once if another change lands first, so it must only set fields.
func Update(fn func(*Options)) {
	for {
		old := current.Load()
		next := clone(old)
		fn(next)
		if current.CompareAndSwap(old, next) {
			return
		}
	}
}

// SnapshotConfig returns a function that restores the configuration to its current state.
func SnapshotConfig() func() {
	snapshot := current.Load()
	return func() { current.Store(snapshot) }
}

func clone(o *Options) *Options {
	var c Options
	deepCopy(reflect.ValueOf(&c).Elem(), reflect.ValueOf(o).Elem())
	return &c
}

func deepCopy(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Struct:
		dst.Set(src)
		for i := range src.NumField() {
			if dst.Field(i).CanSet() {
				deepCopy(dst.Field(i), src.Field(i))
			}
		}
	case reflect.Map:
		if src.IsNil() {
			dst.Set(src)
			return
		}
		m := reflect.MakeMapWithSize(src.Type(), src.Len())
		for k, v := range src.Seq2() {
			nv := reflect.New(v.Type()).Elem()
			deepCopy(nv, v)
			m.SetMapIndex(k, nv)
		}
		dst.Set(m)
	case reflect.Slice:
		if src.IsNil() {
			dst.Set(src)
			return
		}
		s := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
		for i := range src.Len() {
			deepCopy(s.Index(i), src.Index(i))
		}
		dst.Set(s)
	case reflect.Pointer:
		if src.IsNil() {
			dst.Set(src)
			return
		}
		p := reflect.New(src.Type().Elem())
		deepCopy(p.Elem(), src.Elem())
		dst.Set(p)
	default:
		dst.Set(src)
	}
}
