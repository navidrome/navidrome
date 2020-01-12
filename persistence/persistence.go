package persistence

import "reflect"

type ProviderIdentifier string

func CollectValue(collection interface{}, getValue func(item interface{}) string) []string {
	s := reflect.ValueOf(collection)
	result := make([]string, s.Len())

	for i := 0; i < s.Len(); i++ {
		result[i] = getValue(s.Index(i).Interface())
	}

	return result
}
