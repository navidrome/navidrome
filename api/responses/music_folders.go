package responses

import "encoding/xml"

type MusicFolder struct {
	XMLName xml.Name `xml:"musicFolder"`
	Id      string `xml:"id,attr"`
	Name    string `xml:"name,attr"`
}

type MusicFolders struct {
	XMLName xml.Name `xml:"musicFolders"`
	Folders []MusicFolder `xml:"musicFolders"`
}