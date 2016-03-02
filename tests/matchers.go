package tests

import (
	. "github.com/smartystreets/goconvey/convey"
	"encoding/json"
	"encoding/xml"
	"fmt"
)

func ShouldMatchXML(actual interface{}, expected ...interface{}) string {
	xml, err := xml.Marshal(actual)
	if err != nil {
		return fmt.Sprintf("Malformed XML: %v", err)
	}
	return ShouldEqual(string(xml), expected[0].(string))

}

func ShouldMatchJSON(actual interface{}, expected ...interface{}) string {
	json, err := json.Marshal(actual)
	if err != nil {
		return fmt.Sprintf("Malformed JSON: %v", err)
	}
	s := UnindentJSON(json)
	return ShouldEqual(s, expected[0].(string))
}

func UnindentJSON(j []byte) string {
	var m = make(map[string]interface{})
	json.Unmarshal(j, &m)
	s, _ := json.Marshal(m)
	return string(s)
}