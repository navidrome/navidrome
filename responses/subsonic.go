package responses

import (
	"encoding/xml"
	"github.com/astaxie/beego"
)

type Subsonic struct {
	XMLName xml.Name `xml:"http://subsonic.org/restapi subsonic-response"`
	Status  string   `xml:"status,attr"`
	Version string   `xml:"version,attr"`
}

func NewSubsonic() Subsonic {
	return Subsonic{Status: "ok", Version: beego.AppConfig.String("apiversion")}
}