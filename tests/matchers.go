package tests

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/deluan/gosonic/api/responses"
	. "github.com/smartystreets/goconvey/convey"
	"crypto/md5"
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

func ShouldContainJSON(actual interface{}, expected ...interface{}) string {
	a := UnindentJSON(actual.(*bytes.Buffer).Bytes())

	return ShouldContainSubstring(a, expected[0].(string))
}

func ShouldReceiveError(actual interface{}, expected ...interface{}) string {
	v := responses.Subsonic{}
	err := xml.Unmarshal(actual.(*bytes.Buffer).Bytes(), &v)
	if err != nil {
		return fmt.Sprintf("Malformed XML: %v", err)
	}

	return ShouldEqual(v.Error.Code, expected[0].(int))
}

func ShouldMatchMD5(actual interface{}, expected ...interface{}) string {
	a := fmt.Sprintf("%x", md5.Sum(actual.([]byte)))
	return ShouldEqual(a, expected[0].(string))
}

func UnindentJSON(j []byte) string {
	var m = make(map[string]interface{})
	json.Unmarshal(j, &m)
	s, _ := json.Marshal(m)
	return string(s)
}
