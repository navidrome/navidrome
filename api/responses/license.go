package responses

import "encoding/xml"

type License struct {
	XMLName xml.Name `xml:"license"`
	Valid   bool     `xml:"valid,attr"`
}
