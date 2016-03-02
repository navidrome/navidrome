package responses

import "encoding/xml"

type Subsonic struct {
	XMLName      xml.Name `xml:"http://subsonic.org/restapi subsonic-response"`
	Status       string   `xml:"status,attr"`
	Version      string   `xml:"version,attr"`
	Body         []byte   `xml:",innerxml"`
	License      License  `xml:",omitempty"`
	MusicFolders MusicFolders `xml:",omitempty"`
	ArtistIndex  ArtistIndex `xml:",omitempty"`
}

type License struct {
	XMLName xml.Name `xml:"license"`
	Valid   bool     `xml:"valid,attr"`
}

type MusicFolder struct {
	XMLName xml.Name `xml:"musicFolder"`
	Id      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
}

type MusicFolders struct {
	XMLName xml.Name      `xml:"musicFolders"`
	Folders []MusicFolder `xml:"musicFolders"`
}

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
	LastModified    string     `xml:"lastModified,attr"`
	IgnoredArticles string     `xml:"ignoredArticles,attr"`
}

