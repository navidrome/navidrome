package db_ledis

import (
	"encoding/json"
)

func toMap(rec interface{}) (map[string]interface{}, error) {
	// Convert to JSON...
	b, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}

	// ... then convert to map
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	return m, err
}

func toStruct(m map[string]interface{}, rec interface{}) error {
	// Convert to JSON...
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	// ... then convert to struct
	err = json.Unmarshal(b, &rec)
	return err
}
