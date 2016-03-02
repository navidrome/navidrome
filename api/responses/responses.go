package responses

import "encoding/xml"

type Subsonic struct {
	XMLName      xml.Name `xml:"http://subsonic.org/restapi subsonic-response" json:"-"`
	Status       string   `xml:"status,attr" json:"status"`
	Version      string   `xml:"version,attr" json:"version"`
	Error        *Error    `xml:",omitempty"    json:"error,omitempty"`
	License      *License  `xml:",omitempty"   json:"license,omitempty"`
	MusicFolders *MusicFolders `xml:",omitempty"    json:"musicFolders,omitempty"`
	ArtistIndex  *Indexes `xml:",omitempty"    json:"indexes,omitempty"`
}

type Error struct {
	XMLName xml.Name `xml:"error" json:"-"`
	Code    int      `xml:"code,attr"`
	Message string   `xml:"message,attr"`
}

type License struct {
	XMLName xml.Name `xml:"license" json:"-" json:"-"`
	Valid   bool     `xml:"valid,attr" json:"valid"`
}

type MusicFolder struct {
	XMLName xml.Name `xml:"musicFolder" json:"-"`
	Id      string   `xml:"id,attr" json:"id"`
	Name    string   `xml:"name,attr" json:"name"`
}

type MusicFolders struct {
	XMLName xml.Name      `xml:"musicFolders" json:"-"`
	Folders []MusicFolder `xml:"musicFolders" json:"musicFolder"`
}

type Artist struct {
	XMLName xml.Name `xml:"artist" json:"-"`
	Id      string   `xml:"id,attr" json:"id"`
	Name    string   `xml:"name,attr" json:"name"`
}

type Index struct {
	XMLName xml.Name    `xml:"index" json:"-"`
	Name    string      `xml:"name,attr" json:"name"`
	Artists []Artist `xml:"index" json:"artist"`
}

type Indexes struct {
	XMLName         xml.Name   `xml:"indexes" json:"-"`
	Index           []Index `xml:"indexes" json:"index"`
	LastModified    string     `xml:"lastModified,attr" json:"lastModified"`
	IgnoredArticles string     `xml:"ignoredArticles,attr" json:"ignoredArticles"`
}

