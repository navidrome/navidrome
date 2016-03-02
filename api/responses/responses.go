package responses

import "encoding/xml"

type Subsonic struct {
	XMLName      xml.Name      `xml:"http://subsonic.org/restapi subsonic-response" json:"-"`
	Status       string        `xml:"status,attr"                                   json:"status"`
	Version      string        `xml:"version,attr"                                  json:"version"`
	Error        *Error        `xml:",omitempty"                                    json:"error,omitempty"`
	License      *License      `xml:"license,omitempty"                             json:"license,omitempty"`
	MusicFolders *MusicFolders `xml:"musicFolders,omitempty"                        json:"musicFolders,omitempty"`
	Indexes      *Indexes      `xml:"indexes,omitempty"                             json:"indexes,omitempty"`
	Directory    *Directory    `xml:"directory,omitempty"                           json:"directory,omitempty"`
}

type JsonWrapper struct {
	Subsonic Subsonic `json:"subsonic-response"`
}

type Error struct {
	Code    int      `xml:"code,attr"`
	Message string   `xml:"message,attr"`
}

type License struct {
	Valid   bool     `xml:"valid,attr"                    json:"valid"`
}

type MusicFolder struct {
	Id      string   `xml:"id,attr"                       json:"id"`
	Name    string   `xml:"name,attr"                     json:"name"`
}

type MusicFolders struct {
	Folders []MusicFolder `xml:"musicFolder"              json:"musicFolder,omitempty"`
}

type Artist struct {
	Id      string   `xml:"id,attr"                       json:"id"`
	Name    string   `xml:"name,attr"                     json:"name"`
}

type Index struct {
	Name    string   `xml:"name,attr"                     json:"name"`
	Artists []Artist `xml:"artist"                        json:"artist"`
}

type Indexes struct {
	Index           []Index  `xml:"index"                 json:"index,omitempty"`
	LastModified    string   `xml:"lastModified,attr"     json:"lastModified"`
	IgnoredArticles string   `xml:"ignoredArticles,attr"  json:"ignoredArticles"`
}

type Child struct {
	Id string                       `xml:"id,attr"                       json:"id"`
	IsDir bool                      `xml:"isDir,attr"                    json:"isDir"`
	Title string                    `xml:"title,attr"                    json:"title"`
	Album string                    `xml:"album,attr"                    json:"album"`
	Artist string                   `xml:"artist,attr"                   json:"artist"`
	Track int                       `xml:"track,attr"                    json:"track"`
	Year int                        `xml:"year,attr"                     json:"year"`
	Genre string                    `xml:"genre,attr"                    json:"genre"`
	CoverArt string                 `xml:"coverArt,attr"                 json:"coverArt"`
	Size string                     `xml:"size,attr"                     json:"size"`
	ContentType string              `xml:"contentType,attr"              json:"contentType"`
	Suffix string                   `xml:"suffix,attr"                   json:"suffix"`
	TranscodedContentType string    `xml:"transcodedContentType,attr"    json:"transcodedContentType"`
	TranscodedSuffix string         `xml:"transcodedSuffix,attr"         json:"transcodedSuffix"`
	Duration int                    `xml:"duration,attr"                 json:"duration"`
	BitRate int                     `xml:"bitRate,attr"                  json:"bitRate"`
}

type Directory struct {
	Child []Child     `xml:"child"                         json:"child,omitempty"`
	Id string         `xml:"id,attr"                       json:"id"`
	Name string       `xml:"name,attr"                     json:"name"`
}