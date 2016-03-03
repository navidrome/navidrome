package responses

import "encoding/xml"

type Subsonic struct {
	XMLName      xml.Name      `xml:"http://subsonic.org/restapi subsonic-response" json:"-"`
	Status       string        `xml:"status,attr"                                   json:"status"`
	Version      string        `xml:"version,attr"                                  json:"version"`
	Error        *Error        `xml:"error,omitempty"                               json:"error,omitempty"`
	License      *License      `xml:"license,omitempty"                             json:"license,omitempty"`
	MusicFolders *MusicFolders `xml:"musicFolders,omitempty"                        json:"musicFolders,omitempty"`
	Indexes      *Indexes      `xml:"indexes,omitempty"                             json:"indexes,omitempty"`
	Directory    *Directory    `xml:"directory,omitempty"                           json:"directory,omitempty"`
}

type JsonWrapper struct {
	Subsonic Subsonic `json:"subsonic-response"`
}

type Error struct {
	Code    int      `xml:"code,attr"                     json:"code"`
	Message string   `xml:"message,attr"                  json: "message"`
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
	Id string                       `xml:"id,attr"                                 json:"id"`
	IsDir bool                      `xml:"isDir,attr"                              json:"isDir"`
	Title string                    `xml:"title,attr"                              json:"title"`
	Album string                    `xml:"album,attr,omitempty"                    json:"album,omitempty"`
	Artist string                   `xml:"artist,attr,omitempty"                   json:"artist,omitempty"`
	Track int                       `xml:"track,attr,omitempty"                    json:"track,omitempty"`
	Year int                        `xml:"year,attr,omitempty"                     json:"year,omitempty"`
	Genre string                    `xml:"genre,attr,omitempty"                    json:"genre,omitempty"`
	CoverArt string                 `xml:"coverArt,attr,omitempty"                 json:"coverArt,omitempty"`
	Size string                     `xml:"size,attr,omitempty"                     json:"size,omitempty"`
	ContentType string              `xml:"contentType,attr,omitempty"              json:"contentType,omitempty"`
	Suffix string                   `xml:"suffix,attr,omitempty"                   json:"suffix,omitempty"`
	TranscodedContentType string    `xml:"transcodedContentType,attr,omitempty"    json:"transcodedContentType,omitempty"`
	TranscodedSuffix string         `xml:"transcodedSuffix,attr,omitempty"         json:"transcodedSuffix,omitempty"`
	Duration int                    `xml:"duration,attr,omitempty"                 json:"duration,omitempty"`
	BitRate int                     `xml:"bitRate,attr,omitempty"                  json:"bitRate,omitempty"`
}

type Directory struct {
	Child []Child     `xml:"child"                         json:"child,omitempty"`
	Id string         `xml:"id,attr"                       json:"id"`
	Name string       `xml:"name,attr"                     json:"name"`
}