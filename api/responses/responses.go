package responses

import (
	"encoding/xml"
	"time"
)

type Subsonic struct {
	XMLName      xml.Name           `xml:"http://subsonic.org/restapi subsonic-response" json:"-"`
	Status       string             `xml:"status,attr"                                   json:"status"`
	Version      string             `xml:"version,attr"                                  json:"version"`
	Error        *Error             `xml:"error,omitempty"                               json:"error,omitempty"`
	License      *License           `xml:"license,omitempty"                             json:"license,omitempty"`
	MusicFolders *MusicFolders      `xml:"musicFolders,omitempty"                        json:"musicFolders,omitempty"`
	Indexes      *Indexes           `xml:"indexes,omitempty"                             json:"indexes,omitempty"`
	Directory    *Directory         `xml:"directory,omitempty"                           json:"directory,omitempty"`
	User         *User              `xml:"user,omitempty"                                json:"user,omitempty"`
	AlbumList    *AlbumList         `xml:"albumList,omitempty"                           json:"albumList,omitempty"`
	Playlists    *Playlists         `xml:"playlists,omitempty"                           json:"playlists,omitempty"`
	Playlist     *PlaylistWithSongs `xml:"playlist,omitempty"                            json:"playlist,omitempty"`
}

type JsonWrapper struct {
	Subsonic Subsonic `json:"subsonic-response"`
}

type Error struct {
	Code    int    `xml:"code,attr"                     json:"code"`
	Message string `xml:"message,attr"                  json:"message"`
}

type License struct {
	Valid bool `xml:"valid,attr"                    json:"valid"`
}

type MusicFolder struct {
	Id   string `xml:"id,attr"                       json:"id"`
	Name string `xml:"name,attr"                     json:"name"`
}

type MusicFolders struct {
	Folders []MusicFolder `xml:"musicFolder"              json:"musicFolder,omitempty"`
}

type Artist struct {
	Id   string `xml:"id,attr"                       json:"id"`
	Name string `xml:"name,attr"                     json:"name"`
}

type Index struct {
	Name    string   `xml:"name,attr"                     json:"name"`
	Artists []Artist `xml:"artist"                        json:"artist"`
}

type Indexes struct {
	Index           []Index `xml:"index"                 json:"index,omitempty"`
	LastModified    string  `xml:"lastModified,attr"     json:"lastModified"`
	IgnoredArticles string  `xml:"ignoredArticles,attr"  json:"ignoredArticles"`
}

type Child struct {
	Id                    string     `xml:"id,attr"                                 json:"id"`
	Parent                string     `xml:"parent,attr,omitempty"                   json:"parent,omitempty"`
	IsDir                 bool       `xml:"isDir,attr"                              json:"isDir"`
	Title                 string     `xml:"title,attr"                              json:"title"`
	Album                 string     `xml:"album,attr,omitempty"                    json:"album,omitempty"`
	Artist                string     `xml:"artist,attr,omitempty"                   json:"artist,omitempty"`
	Track                 int        `xml:"track,attr,omitempty"                    json:"track,omitempty"`
	Year                  int        `xml:"year,attr,omitempty"                     json:"year,omitempty"`
	Genre                 string     `xml:"genre,attr,omitempty"                    json:"genre,omitempty"`
	CoverArt              string     `xml:"coverArt,attr,omitempty"                 json:"coverArt,omitempty"`
	Size                  string     `xml:"size,attr,omitempty"                     json:"size,omitempty"`
	ContentType           string     `xml:"contentType,attr,omitempty"              json:"contentType,omitempty"`
	Suffix                string     `xml:"suffix,attr,omitempty"                   json:"suffix,omitempty"`
	Starred               *time.Time `xml:"starred,attr,omitempty"                  json:"starred,omitempty"`
	TranscodedContentType string     `xml:"transcodedContentType,attr,omitempty"    json:"transcodedContentType,omitempty"`
	TranscodedSuffix      string     `xml:"transcodedSuffix,attr,omitempty"         json:"transcodedSuffix,omitempty"`
	Duration              int        `xml:"duration,attr,omitempty"                 json:"duration,omitempty"`
	BitRate               int        `xml:"bitRate,attr,omitempty"                  json:"bitRate,omitempty"`
}

type Directory struct {
	Child []Child `xml:"child"                         json:"child,omitempty"`
	Id    string  `xml:"id,attr"                       json:"id"`
	Name  string  `xml:"name,attr"                     json:"name"`
}

type AlbumList struct {
	Album []Child `xml:"album"                         json:"album,omitempty"`
}

type Playlist struct {
	Id        string `xml:"id,attr"                                 json:"id"`
	Name      string `xml:"name,attr"                               json:"name"`
	Comment   string `xml:"comment,attr,omitempty"                  json:"comment,omitempty"`
	SongCount int    `xml:"songCount,attr,omitempty"                json:"songCount,omitempty"`
	/*
		<xs:sequence>
		    <xs:element name="allowedUser" type="xs:string" minOccurs="0" maxOccurs="unbounded"/> <!--Added in 1.8.0-->
		</xs:sequence>
		<xs:attribute name="comment" type="xs:string" use="optional"/>   <!--Added in 1.8.0-->
		<xs:attribute name="owner" type="xs:string" use="optional"/>     <!--Added in 1.8.0-->
		<xs:attribute name="public" type="xs:boolean" use="optional"/>   <!--Added in 1.8.0-->
		<xs:attribute name="songCount" type="xs:int" use="required"/>    <!--Added in 1.8.0-->
		<xs:attribute name="duration" type="xs:int" use="required"/>     <!--Added in 1.8.0-->
		<xs:attribute name="created" type="xs:dateTime" use="required"/> <!--Added in 1.8.0-->
		<xs:attribute name="changed" type="xs:dateTime" use="required"/> <!--Added in 1.13.0-->
		<xs:attribute name="coverArt" type="xs:string" use="optional"/>  <!--Added in 1.11.0-->

	*/
}

type Playlists struct {
	Playlist []Playlist `xml:"playlist"                         json:"playlist,omitempty"`
}

type PlaylistWithSongs struct {
	Playlist
	Entry []Child `xml:"entry"                            json:"entry,omitempty"`
}

type User struct {
	Username            string `xml:"username,attr" json:"username"`
	Email               string `xml:"email,attr,omitempty" json:"email,omitempty"`
	ScrobblingEnabled   bool   `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	MaxBitRate          int    `xml:"maxBitRate,attr,omitempty" json:"maxBitRate,omitempty"`
	AdminRole           bool   `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole        bool   `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole        bool   `xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole          bool   `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole        bool   `xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole        bool   `xml:"coverArtRole,attr" json:"coverArtRole"`
	CommentRole         bool   `xml:"commentRole,attr" json:"commentRole"`
	PodcastRole         bool   `xml:"podcastRole,attr" json:"podcastRole"`
	StreamRole          bool   `xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole         bool   `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	ShareRole           bool   `xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool   `xml:"videoConversionRole,attr" json:"videoConversionRole"`
	Folder              []int  `xml:"folder,omitempty" json:"folder,omitempty"`
}
