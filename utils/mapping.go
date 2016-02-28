// Adapted from https://github.com/nytlabs/gojsonexplode

package utils

import (
	"strconv"
	"encoding/json"
)

const delimiter = "."

func flattenList(l []interface{}, parent string, delimiter string) (map[string]interface{}, error) {
	var err error
	var key string
	j := make(map[string]interface{})
	for k, i := range l {
		if len(parent) > 0 {
			key = parent + delimiter + strconv.Itoa(k)
		} else {
			key = strconv.Itoa(k)
		}
		switch v := i.(type) {
		case nil:
			j[key] = v
		case int:
			j[key] = v
		case float64:
			j[key] = v
		case string:
			j[key] = v
		case bool:
			j[key] = v
		case []interface{}:
			out := make(map[string]interface{})
			out, err = flattenList(v, key, delimiter)
			if err != nil {
				return nil, err
			}
			for newkey, value := range out {
				j[newkey] = value
			}
		case map[string]interface{}:
			out := make(map[string]interface{})
			out, err = flattenMap(v, key, delimiter)
			if err != nil {
				return nil, err
			}
			for newkey, value := range out {
				j[newkey] = value
			}
		default:
		// do nothing
		}
	}
	return j, nil
}

func flattenMap(m map[string]interface{}, parent string, delimiter string) (map[string]interface{}, error) {
	var err error
	j := make(map[string]interface{})
	for k, i := range m {
		if len(parent) > 0 {
			k = parent + delimiter + k
		}
		switch v := i.(type) {
		case nil:
			j[k] = v
		case int:
			j[k] = v
		case float64:
			j[k] = v
		case string:
			j[k] = v
		case bool:
			j[k] = v
		case []interface{}:
			out := make(map[string]interface{})
			out, err = flattenList(v, k, delimiter)
			if err != nil {
				return nil, err
			}
			for key, value := range out {
				j[key] = value
			}
		case map[string]interface{}:
			out := make(map[string]interface{})
			out, err = flattenMap(v, k, delimiter)
			if err != nil {
				return nil, err
			}
			for key, value := range out {
				j[key] = value
			}
		default:
		//nothing
		}
	}
	return j, nil
}

func FlattenMap(input map[string]interface{}) (map[string]interface{}, error) {
	var flattened map[string]interface{}
	var err error
	flattened, err = flattenMap(input, "", delimiter)
	if err != nil {
		return nil, err
	}
	return flattened, nil
}

func ToMap(rec interface{}) (map[string]interface{}, error) {
	// Convert to JSON...
	b, err := json.Marshal(rec);
	if err != nil {
		return nil, err
	}

	// ... then convert to map
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	return m, err
}

func ToStruct(m map[string]interface{}, rec interface{}) error {
	// Convert to JSON...
	b, err := json.Marshal(m);
	if err != nil {
		return err
	}

	// ... then convert to struct
	err = json.Unmarshal(b, &rec)
	return err
}

func Flatten(input interface{}) (map[string]interface{}, error) {
	m, err := ToMap(input)
	if err != nil {
		return nil, err
	}
	return FlattenMap(m)
}
