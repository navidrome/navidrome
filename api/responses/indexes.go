package responses

import "encoding/xml"

type Index struct {
	XMLName xml.Name  `xml:"index"`
	Name    string    `xml:"name,attr"`
}

type ArtistIndex struct {
	XMLName xml.Name  `xml:"indexes"`
	Index   []Index   `xml:"indexes"`
}

