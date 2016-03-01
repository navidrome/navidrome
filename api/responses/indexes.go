package responses

import "encoding/xml"

type IdxArtist struct {
	XMLName xml.Name `xml:"artist"`
	Id      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
}

type IdxIndex struct {
	XMLName xml.Name    `xml:"index"`
	Name    string      `xml:"name,attr"`
	Artists []IdxArtist `xml:"index"`
}

type ArtistIndex struct {
	XMLName         xml.Name   `xml:"indexes"`
	Index           []IdxIndex `xml:"indexes"`
	IgnoredArticles string     `xml:"ignoredArticles,attr"`
}

