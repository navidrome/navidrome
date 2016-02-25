package responses

import (
	"encoding/xml"
	"github.com/astaxie/beego"
)

type Subsonic struct {
	XMLName xml.Name `xml:"http://subsonic.org/restapi subsonic-response"`
	Status  string   `xml:"status,attr"`
	Version string   `xml:"version,attr"`
	Body    []byte   `xml:",innerxml"`
}

func NewEmpty() Subsonic {
	return Subsonic{Status: "ok", Version: beego.AppConfig.String("apiVersion")}
}

func NewXML(body interface{}) []byte {
	response := NewEmpty()
	xmlBody, _ := xml.Marshal(body)
	response.Body = xmlBody
	xmlResponse, _ := xml.Marshal(response)
	return []byte(xml.Header + string(xmlResponse))
}
