package storm

import (
	"reflect"
)

func tag(entity interface{}) interface{} {
	st := reflect.TypeOf(entity).Elem()
	var fs []reflect.StructField
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		f.Tag = mapTags(f.Tag)
		fs = append(fs, f)
	}

	st2 := reflect.StructOf(fs)
	v := reflect.ValueOf(entity).Elem()
	v2 := v.Convert(st2)
	vp := reflect.New(st2)
	vp.Elem().Set(reflect.ValueOf(v2.Interface()))
	return vp.Interface()
}

func mapTags(tags reflect.StructTag) reflect.StructTag {
	if tags == `db:"index"` {
		return `storm:"index"`
	}
	return tags
}

func getTypeName(myVar interface{}) string {
	if t := reflect.TypeOf(myVar); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

func getStructTag(instance interface{}, fieldName string) string {
	field, _ := reflect.TypeOf(instance).Elem().FieldByName(fieldName)
	return string(field.Tag)
}
