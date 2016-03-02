package responses

import (
	"encoding/xml"
	"github.com/astaxie/beego"
)

func NewEmpty() Subsonic {
	return Subsonic{Status: "ok", Version: beego.AppConfig.String("apiVersion")}
}

func ToXML(response Subsonic) []byte {
	xmlBody, _ := xml.Marshal(response)
	return []byte(xml.Header + string(xmlBody))
}
