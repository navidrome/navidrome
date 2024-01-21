package responses

import (
	"encoding/json"
	"encoding/xml"
	"time"
)

type Subsonic struct {
	XMLName       xml.Name           `xml:"http://subsonic.org/restapi subsonic-response" json:"-"`
	Status        string             `xml:"status,attr"                                   json:"status"`
	Version       string             `xml:"version,attr"                                  json:"version"`
	Type          string             `xml:"type,attr"                                     json:"type"`
	ServerVersion string             `xml:"serverVersion,attr"                            json:"serverVersion"`
	OpenSubsonic  bool               `xml:"openSubsonic,attr,omitempty"                   json:"openSubsonic,omitempty"`
	Error         *Error             `xml:"error,omitempty"                               json:"error,omitempty"`
	License       *License           `xml:"license,omitempty"                             json:"license,omitempty"`
	MusicFolders  *MusicFolders      `xml:"musicFolders,omitempty"                        json:"musicFolders,omitempty"`
	Indexes       *Indexes           `xml:"indexes,omitempty"                             json:"indexes,omitempty"`
	Directory     *Directory         `xml:"directory,omitempty"                           json:"directory,omitempty"`
	User          *User              `xml:"user,omitempty"                                json:"user,omitempty"`
	Users         *Users             `xml:"users,omitempty"                               json:"users,omitempty"`
	AlbumList     *AlbumList         `xml:"albumList,omitempty"                           json:"albumList,omitempty"`
	AlbumList2    *AlbumList         `xml:"albumList2,omitempty"                          json:"albumList2,omitempty"`
	Playlists     *Playlists         `xml:"playlists,omitempty"                           json:"playlists,omitempty"`
	Playlist      *PlaylistWithSongs `xml:"playlist,omitempty"                            json:"playlist,omitempty"`
	SearchResult2 *SearchResult2     `xml:"searchResult2,omitempty"                       json:"searchResult2,omitempty"`
	SearchResult3 *SearchResult3     `xml:"searchResult3,omitempty"                       json:"searchResult3,omitempty"`
	Starred       *Starred           `xml:"starred,omitempty"                             json:"starred,omitempty"`
	Starred2      *Starred           `xml:"starred2,omitempty"                            json:"starred2,omitempty"`
	NowPlaying    *NowPlaying        `xml:"nowPlaying,omitempty"                          json:"nowPlaying,omitempty"`
	Song          *Child             `xml:"song,omitempty"                                json:"song,omitempty"`
	RandomSongs   *Songs             `xml:"randomSongs,omitempty"                         json:"randomSongs,omitempty"`
	SongsByGenre  *Songs             `xml:"songsByGenre,omitempty"                        json:"songsByGenre,omitempty"`
	Genres        *Genres            `xml:"genres,omitempty"                              json:"genres,omitempty"`

	// ID3
	Artist              *Indexes             `xml:"artists,omitempty"                     json:"artists,omitempty"`
	ArtistWithAlbumsID3 *ArtistWithAlbumsID3 `xml:"artist,omitempty"                      json:"artist,omitempty"`
	AlbumWithSongsID3   *AlbumWithSongsID3   `xml:"album,omitempty"                       json:"album,omitempty"`

	AlbumInfo     *AlbumInfo     `xml:"albumInfo,omitempty"                               json:"albumInfo,omitempty"`
	ArtistInfo    *ArtistInfo    `xml:"artistInfo,omitempty"                              json:"artistInfo,omitempty"`
	ArtistInfo2   *ArtistInfo2   `xml:"artistInfo2,omitempty"                             json:"artistInfo2,omitempty"`
	SimilarSongs  *SimilarSongs  `xml:"similarSongs,omitempty"                            json:"similarSongs,omitempty"`
	SimilarSongs2 *SimilarSongs2 `xml:"similarSongs2,omitempty"                           json:"similarSongs2,omitempty"`
	TopSongs      *TopSongs      `xml:"topSongs,omitempty"                                json:"topSongs,omitempty"`

	PlayQueue  *PlayQueue  `xml:"playQueue,omitempty"                                     json:"playQueue,omitempty"`
	Shares     *Shares     `xml:"shares,omitempty"                                     json:"shares,omitempty"`
	Bookmarks  *Bookmarks  `xml:"bookmarks,omitempty"                                     json:"bookmarks,omitempty"`
	ScanStatus *ScanStatus `xml:"scanStatus,omitempty"                                    json:"scanStatus,omitempty"`
	Lyrics     *Lyrics     `xml:"lyrics,omitempty"                                        json:"lyrics,omitempty"`

	InternetRadioStations *InternetRadioStations `xml:"internetRadioStations,omitempty"   json:"internetRadioStations,omitempty"`

	JukeboxStatus   *JukeboxStatus   `xml:"jukeboxStatus,omitempty"                       json:"jukeboxStatus,omitempty"`
	JukeboxPlaylist *JukeboxPlaylist `xml:"jukeboxPlaylist,omitempty"                     json:"jukeboxPlaylist,omitempty"`

	OpenSubsonicExtensions *OpenSubsonicExtensions `xml:"openSubsonicExtensions,omitempty"  json:"openSubsonicExtensions,omitempty"`
	LyricsList             *LyricsList             `xml:"lyricsList,omitempty" json:"lyricsList,omitempty"`
}

type JsonWrapper struct {
	Subsonic Subsonic `json:"subsonic-response"`
}

type Error struct {
	Code    int32  `xml:"code,attr"                      json:"code"`
	Message string `xml:"message,attr"                   json:"message"`
}

type License struct {
	Valid bool `xml:"valid,attr"                         json:"valid"`
}

type MusicFolder struct {
	Id   int32  `xml:"id,attr"                           json:"id"`
	Name string `xml:"name,attr"                         json:"name"`
}

type MusicFolders struct {
	Folders []MusicFolder `xml:"musicFolder"             json:"musicFolder,omitempty"`
}

type Artist struct {
	Id             string     `xml:"id,attr"                           json:"id"`
	Name           string     `xml:"name,attr"                         json:"name"`
	AlbumCount     int32      `xml:"albumCount,attr,omitempty"         json:"albumCount,omitempty"`
	Starred        *time.Time `xml:"starred,attr,omitempty"            json:"starred,omitempty"`
	UserRating     int32      `xml:"userRating,attr,omitempty"         json:"userRating,omitempty"`
	CoverArt       string     `xml:"coverArt,attr,omitempty"           json:"coverArt,omitempty"`
	ArtistImageUrl string     `xml:"artistImageUrl,attr,omitempty"     json:"artistImageUrl,omitempty"`
	/* TODO:
	<xs:attribute name="averageRating" type="sub:AverageRating" use="optional"/>  <!-- Added in 1.13.0 -->
	*/
}

type Index struct {
	Name    string   `xml:"name,attr"                     json:"name"`
	Artists []Artist `xml:"artist"                        json:"artist"`
}

type Indexes struct {
	Index           []Index `xml:"index"                  json:"index,omitempty"`
	LastModified    int64   `xml:"lastModified,attr"      json:"lastModified"`
	IgnoredArticles string  `xml:"ignoredArticles,attr"   json:"ignoredArticles"`
}

type MediaType string

const (
	MediaTypeSong   MediaType = "song"
	MediaTypeAlbum  MediaType = "album"
	MediaTypeArtist MediaType = "artist"
)

type Child struct {
	Id                    string     `xml:"id,attr"                                 json:"id"`
	Parent                string     `xml:"parent,attr,omitempty"                   json:"parent,omitempty"`
	IsDir                 bool       `xml:"isDir,attr"                              json:"isDir"`
	Title                 string     `xml:"title,attr,omitempty"                    json:"title,omitempty"`
	Name                  string     `xml:"name,attr,omitempty"                     json:"name,omitempty"`
	Album                 string     `xml:"album,attr,omitempty"                    json:"album,omitempty"`
	Artist                string     `xml:"artist,attr,omitempty"                   json:"artist,omitempty"`
	Track                 int32      `xml:"track,attr,omitempty"                    json:"track,omitempty"`
	Year                  int32      `xml:"year,attr,omitempty"                     json:"year,omitempty"`
	Genre                 string     `xml:"genre,attr,omitempty"                    json:"genre,omitempty"`
	CoverArt              string     `xml:"coverArt,attr,omitempty"                 json:"coverArt,omitempty"`
	Size                  int64      `xml:"size,attr,omitempty"                     json:"size,omitempty"`
	ContentType           string     `xml:"contentType,attr,omitempty"              json:"contentType,omitempty"`
	Suffix                string     `xml:"suffix,attr,omitempty"                   json:"suffix,omitempty"`
	Starred               *time.Time `xml:"starred,attr,omitempty"                  json:"starred,omitempty"`
	TranscodedContentType string     `xml:"transcodedContentType,attr,omitempty"    json:"transcodedContentType,omitempty"`
	TranscodedSuffix      string     `xml:"transcodedSuffix,attr,omitempty"         json:"transcodedSuffix,omitempty"`
	Duration              int32      `xml:"duration,attr,omitempty"                 json:"duration,omitempty"`
	BitRate               int32      `xml:"bitRate,attr,omitempty"                  json:"bitRate,omitempty"`
	Path                  string     `xml:"path,attr,omitempty"                     json:"path,omitempty"`
	PlayCount             int64      `xml:"playCount,attr,omitempty"                json:"playCount,omitempty"`
	DiscNumber            int32      `xml:"discNumber,attr,omitempty"               json:"discNumber,omitempty"`
	Created               *time.Time `xml:"created,attr,omitempty"                  json:"created,omitempty"`
	AlbumId               string     `xml:"albumId,attr,omitempty"                  json:"albumId,omitempty"`
	ArtistId              string     `xml:"artistId,attr,omitempty"                 json:"artistId,omitempty"`
	Type                  string     `xml:"type,attr,omitempty"                     json:"type,omitempty"`
	UserRating            int32      `xml:"userRating,attr,omitempty"               json:"userRating,omitempty"`
	SongCount             int32      `xml:"songCount,attr,omitempty"                json:"songCount,omitempty"`
	IsVideo               bool       `xml:"isVideo,attr"                            json:"isVideo"`
	BookmarkPosition      int64      `xml:"bookmarkPosition,attr,omitempty"         json:"bookmarkPosition,omitempty"`
	/*
	   <xs:attribute name="averageRating" type="sub:AverageRating" use="optional"/>  <!-- Added in 1.6.0 -->
	*/
	// OpenSubsonic extensions
	Played        *time.Time `xml:"played,attr,omitempty"   json:"played,omitempty"`
	Bpm           int32      `xml:"bpm,attr"                json:"bpm"`
	Comment       string     `xml:"comment,attr"            json:"comment"`
	SortName      string     `xml:"sortName,attr"           json:"sortName"`
	MediaType     MediaType  `xml:"mediaType,attr"          json:"mediaType"`
	MusicBrainzId string     `xml:"musicBrainzId,attr"      json:"musicBrainzId"`
	Genres        ItemGenres `xml:"genres"                  json:"genres"`
	ReplayGain    ReplayGain `xml:"replayGain"              json:"replayGain"`
}

type Songs struct {
	Songs []Child `xml:"song"                              json:"song,omitempty"`
}

type Directory struct {
	Child      []Child    `xml:"child"                              json:"child,omitempty"`
	Id         string     `xml:"id,attr"                            json:"id"`
	Name       string     `xml:"name,attr"                          json:"name"`
	Parent     string     `xml:"parent,attr,omitempty"              json:"parent,omitempty"`
	Starred    *time.Time `xml:"starred,attr,omitempty"             json:"starred,omitempty"`
	PlayCount  int64      `xml:"playCount,attr,omitempty"           json:"playCount,omitempty"`
	Played     *time.Time `xml:"played,attr,omitempty"              json:"played,omitempty"`
	UserRating int32      `xml:"userRating,attr,omitempty"          json:"userRating,omitempty"`

	// ID3
	Artist     string     `xml:"artist,attr,omitempty"              json:"artist,omitempty"`
	ArtistId   string     `xml:"artistId,attr,omitempty"            json:"artistId,omitempty"`
	CoverArt   string     `xml:"coverArt,attr,omitempty"            json:"coverArt,omitempty"`
	SongCount  int32      `xml:"songCount,attr,omitempty"           json:"songCount,omitempty"`
	AlbumCount int32      `xml:"albumCount,attr,omitempty"          json:"albumCount,omitempty"`
	Duration   int32      `xml:"duration,attr,omitempty"            json:"duration,omitempty"`
	Created    *time.Time `xml:"created,attr,omitempty"             json:"created,omitempty"`
	Year       int32      `xml:"year,attr,omitempty"                json:"year,omitempty"`
	Genre      string     `xml:"genre,attr,omitempty"               json:"genre,omitempty"`

	/*
	   <xs:attribute name="averageRating" type="sub:AverageRating" use="optional"/>  <!-- Added in 1.13.0 -->
	*/
}

type ArtistID3 struct {
	Id             string     `xml:"id,attr"                            json:"id"`
	Name           string     `xml:"name,attr"                          json:"name"`
	CoverArt       string     `xml:"coverArt,attr,omitempty"            json:"coverArt,omitempty"`
	AlbumCount     int32      `xml:"albumCount,attr,omitempty"          json:"albumCount,omitempty"`
	Starred        *time.Time `xml:"starred,attr,omitempty"             json:"starred,omitempty"`
	UserRating     int32      `xml:"userRating,attr,omitempty"          json:"userRating,omitempty"`
	ArtistImageUrl string     `xml:"artistImageUrl,attr,omitempty"      json:"artistImageUrl,omitempty"`

	// OpenSubsonic extensions
	MusicBrainzId string `xml:"musicBrainzId,attr,omitempty"       json:"musicBrainzId,omitempty"`
	SortName      string `xml:"sortName,attr,omitempty"            json:"sortName,omitempty"`
}

type AlbumID3 struct {
	Id        string     `xml:"id,attr"                            json:"id"`
	Name      string     `xml:"name,attr"                          json:"name"`
	Artist    string     `xml:"artist,attr,omitempty"              json:"artist,omitempty"`
	ArtistId  string     `xml:"artistId,attr,omitempty"            json:"artistId,omitempty"`
	CoverArt  string     `xml:"coverArt,attr,omitempty"            json:"coverArt,omitempty"`
	SongCount int32      `xml:"songCount,attr,omitempty"           json:"songCount,omitempty"`
	Duration  int32      `xml:"duration,attr,omitempty"            json:"duration,omitempty"`
	PlayCount int64      `xml:"playCount,attr,omitempty"           json:"playCount,omitempty"`
	Created   *time.Time `xml:"created,attr,omitempty"             json:"created,omitempty"`
	Starred   *time.Time `xml:"starred,attr,omitempty"             json:"starred,omitempty"`
	Year      int32      `xml:"year,attr,omitempty"                json:"year,omitempty"`
	Genre     string     `xml:"genre,attr,omitempty"               json:"genre,omitempty"`

	// OpenSubsonic extensions
	Played              *time.Time `xml:"played,attr,omitempty" json:"played,omitempty"`
	UserRating          int32      `xml:"userRating,attr"       json:"userRating"`
	Genres              ItemGenres `xml:"genres"                json:"genres"`
	MusicBrainzId       string     `xml:"musicBrainzId,attr"    json:"musicBrainzId"`
	IsCompilation       bool       `xml:"isCompilation,attr"    json:"isCompilation"`
	SortName            string     `xml:"sortName,attr"         json:"sortName"`
	DiscTitles          DiscTitles `xml:"discTitles"            json:"discTitles"`
	OriginalReleaseDate ItemDate   `xml:"originalReleaseDate"   json:"originalReleaseDate"`
}

type ArtistWithAlbumsID3 struct {
	ArtistID3
	Album []Child `xml:"album"                              json:"album,omitempty"`
}

type AlbumWithSongsID3 struct {
	AlbumID3
	Song []Child `xml:"song"                               json:"song,omitempty"`
}

type AlbumList struct {
	Album []Child `xml:"album"                                      json:"album,omitempty"`
}

type Playlist struct {
	Id        string    `xml:"id,attr"                       json:"id"`
	Name      string    `xml:"name,attr"                     json:"name"`
	Comment   string    `xml:"comment,attr,omitempty"        json:"comment,omitempty"`
	SongCount int32     `xml:"songCount,attr"                json:"songCount"`
	Duration  int32     `xml:"duration,attr"                 json:"duration"`
	Public    bool      `xml:"public,attr"                   json:"public"`
	Owner     string    `xml:"owner,attr,omitempty"          json:"owner,omitempty"`
	Created   time.Time `xml:"created,attr"                  json:"created"`
	Changed   time.Time `xml:"changed,attr"                  json:"changed"`
	CoverArt  string    `xml:"coverArt,attr,omitempty"       json:"coverArt,omitempty"`
	/*
		<xs:sequence>
		    <xs:element name="allowedUser" type="xs:string" minOccurs="0" maxOccurs="unbounded"/> <!--Added in 1.8.0-->
		</xs:sequence>
	*/
}

type Playlists struct {
	Playlist []Playlist `xml:"playlist"                           json:"playlist,omitempty"`
}

type PlaylistWithSongs struct {
	Playlist
	Entry []Child `xml:"entry"                                    json:"entry,omitempty"`
}

type SearchResult2 struct {
	Artist []Artist `xml:"artist"                                 json:"artist,omitempty"`
	Album  []Child  `xml:"album"                                  json:"album,omitempty"`
	Song   []Child  `xml:"song"                                   json:"song,omitempty"`
}

type SearchResult3 struct {
	Artist []ArtistID3 `xml:"artist"                                 json:"artist,omitempty"`
	Album  []AlbumID3  `xml:"album"                                  json:"album,omitempty"`
	Song   []Child     `xml:"song"                                   json:"song,omitempty"`
}

type Starred struct {
	Artist []Artist `xml:"artist"                                 json:"artist,omitempty"`
	Album  []Child  `xml:"album"                                  json:"album,omitempty"`
	Song   []Child  `xml:"song"                                   json:"song,omitempty"`
}

type NowPlayingEntry struct {
	Child
	UserName   string `xml:"username,attr"                        json:"username"`
	MinutesAgo int32  `xml:"minutesAgo,attr"                      json:"minutesAgo"`
	PlayerId   int32  `xml:"playerId,attr"                        json:"playerId"`
	PlayerName string `xml:"playerName,attr"                      json:"playerName,omitempty"`
}

type NowPlaying struct {
	Entry []NowPlayingEntry `xml:"entry"                          json:"entry,omitempty"`
}

type User struct {
	Username            string  `xml:"username,attr"               json:"username"`
	Email               string  `xml:"email,attr,omitempty"        json:"email,omitempty"`
	ScrobblingEnabled   bool    `xml:"scrobblingEnabled,attr"      json:"scrobblingEnabled"`
	MaxBitRate          int32   `xml:"maxBitRate,attr,omitempty"   json:"maxBitRate,omitempty"`
	AdminRole           bool    `xml:"adminRole,attr"              json:"adminRole"`
	SettingsRole        bool    `xml:"settingsRole,attr"           json:"settingsRole"`
	DownloadRole        bool    `xml:"downloadRole,attr"           json:"downloadRole"`
	UploadRole          bool    `xml:"uploadRole,attr"             json:"uploadRole"`
	PlaylistRole        bool    `xml:"playlistRole,attr"           json:"playlistRole"`
	CoverArtRole        bool    `xml:"coverArtRole,attr"           json:"coverArtRole"`
	CommentRole         bool    `xml:"commentRole,attr"            json:"commentRole"`
	PodcastRole         bool    `xml:"podcastRole,attr"            json:"podcastRole"`
	StreamRole          bool    `xml:"streamRole,attr"             json:"streamRole"`
	JukeboxRole         bool    `xml:"jukeboxRole,attr"            json:"jukeboxRole"`
	ShareRole           bool    `xml:"shareRole,attr"              json:"shareRole"`
	VideoConversionRole bool    `xml:"videoConversionRole,attr"    json:"videoConversionRole"`
	Folder              []int32 `xml:"folder,omitempty"            json:"folder,omitempty"`
}

type Users struct {
	User []User `xml:"user"  json:"user"`
}

type Genre struct {
	Name       string `xml:",chardata"                      json:"value,omitempty"`
	SongCount  int32  `xml:"songCount,attr"             json:"songCount"`
	AlbumCount int32  `xml:"albumCount,attr"            json:"albumCount"`
}

type Genres struct {
	Genre []Genre `xml:"genre,omitempty"                      json:"genre,omitempty"`
}

type AlbumInfo struct {
	Notes          string `xml:"notes,omitempty"          json:"notes,omitempty"`
	MusicBrainzID  string `xml:"musicBrainzId,omitempty"      json:"musicBrainzId,omitempty"`
	LastFmUrl      string `xml:"lastFmUrl,omitempty"          json:"lastFmUrl,omitempty"`
	SmallImageUrl  string `xml:"smallImageUrl,omitempty"      json:"smallImageUrl,omitempty"`
	MediumImageUrl string `xml:"mediumImageUrl,omitempty"     json:"mediumImageUrl,omitempty"`
	LargeImageUrl  string `xml:"largeImageUrl,omitempty"      json:"largeImageUrl,omitempty"`
}

type ArtistInfoBase struct {
	Biography      string `xml:"biography,omitempty"          json:"biography,omitempty"`
	MusicBrainzID  string `xml:"musicBrainzId,omitempty"      json:"musicBrainzId,omitempty"`
	LastFmUrl      string `xml:"lastFmUrl,omitempty"          json:"lastFmUrl,omitempty"`
	SmallImageUrl  string `xml:"smallImageUrl,omitempty"      json:"smallImageUrl,omitempty"`
	MediumImageUrl string `xml:"mediumImageUrl,omitempty"     json:"mediumImageUrl,omitempty"`
	LargeImageUrl  string `xml:"largeImageUrl,omitempty"      json:"largeImageUrl,omitempty"`
}

type ArtistInfo struct {
	ArtistInfoBase
	SimilarArtist []Artist `xml:"similarArtist,omitempty"    json:"similarArtist,omitempty"`
}

type ArtistInfo2 struct {
	ArtistInfoBase
	SimilarArtist []ArtistID3 `xml:"similarArtist,omitempty"    json:"similarArtist,omitempty"`
}

type SimilarSongs struct {
	Song []Child `xml:"song,omitempty"         json:"song,omitempty"`
}

type SimilarSongs2 struct {
	Song []Child `xml:"song,omitempty"         json:"song,omitempty"`
}

type TopSongs struct {
	Song []Child `xml:"song,omitempty"         json:"song,omitempty"`
}

type PlayQueue struct {
	Entry     []Child    `xml:"entry,omitempty"         json:"entry,omitempty"`
	Current   string     `xml:"current,attr,omitempty"  json:"current,omitempty"`
	Position  int64      `xml:"position,attr,omitempty" json:"position,omitempty"`
	Username  string     `xml:"username,attr"           json:"username"`
	Changed   *time.Time `xml:"changed,attr,omitempty"  json:"changed,omitempty"`
	ChangedBy string     `xml:"changedBy,attr"          json:"changedBy"`
}

type Bookmark struct {
	Entry    Child     `xml:"entry,omitempty"         json:"entry,omitempty"`
	Position int64     `xml:"position,attr,omitempty" json:"position,omitempty"`
	Username string    `xml:"username,attr"           json:"username"`
	Comment  string    `xml:"comment,attr"            json:"comment"`
	Created  time.Time `xml:"created,attr"            json:"created"`
	Changed  time.Time `xml:"changed,attr"            json:"changed"`
}

type Bookmarks struct {
	Bookmark []Bookmark `xml:"bookmark,omitempty"    json:"bookmark,omitempty"`
}

type Share struct {
	Entry       []Child    `xml:"entry,omitempty"             json:"entry,omitempty"`
	ID          string     `xml:"id,attr"                     json:"id"`
	Url         string     `xml:"url,attr"                    json:"url"`
	Description string     `xml:"description,omitempty,attr"  json:"description,omitempty"`
	Username    string     `xml:"username,attr"               json:"username"`
	Created     time.Time  `xml:"created,attr"                json:"created"`
	Expires     *time.Time `xml:"expires,omitempty,attr"      json:"expires,omitempty"`
	LastVisited time.Time  `xml:"lastVisited,omitempty,attr"  json:"lastVisited"`
	VisitCount  int32      `xml:"visitCount,attr"             json:"visitCount"`
}

type Shares struct {
	Share []Share `xml:"share,omitempty" json:"share,omitempty"`
}

type ScanStatus struct {
	Scanning    bool       `xml:"scanning,attr"            json:"scanning"`
	Count       int64      `xml:"count,attr"               json:"count"`
	FolderCount int64      `xml:"folderCount,attr"         json:"folderCount"`
	LastScan    *time.Time `xml:"lastScan,attr,omitempty"  json:"lastScan,omitempty"`
}

type Lyrics struct {
	Artist string `xml:"artist,omitempty,attr"  json:"artist,omitempty"`
	Title  string `xml:"title,omitempty,attr"   json:"title,omitempty"`
	Value  string `xml:",chardata"              json:"value"`
}

type InternetRadioStations struct {
	Radios []Radio `xml:"internetRadioStation"               json:"internetRadioStation,omitempty"`
}

type Radio struct {
	ID          string `xml:"id,attr"                    json:"id"`
	Name        string `xml:"name,attr"                  json:"name"`
	StreamUrl   string `xml:"streamUrl,attr"             json:"streamUrl"`
	HomepageUrl string `xml:"homePageUrl,omitempty,attr" json:"homePageUrl,omitempty"`
}

type JukeboxStatus struct {
	CurrentIndex int32   `xml:"currentIndex,attr"       json:"currentIndex"`
	Playing      bool    `xml:"playing,attr"            json:"playing"`
	Gain         float32 `xml:"gain,attr"               json:"gain"`
	Position     int32   `xml:"position,omitempty,attr" json:"position"`
}

type JukeboxPlaylist struct {
	JukeboxStatus
	Entry []Child `xml:"entry,omitempty"         json:"entry,omitempty"`
}

type Line struct {
	Start *int64 `xml:"start,attr,omitempty" json:"start,omitempty"`
	Value string `xml:"value"                json:"value"`
}

type StructuredLyric struct {
	DisplayArtist string `xml:"displayArtist,attr,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string `xml:"displayTitle,attr,omitempty"  json:"displayTitle,omitempty"`
	Lang          string `xml:"lang,attr"                    json:"lang"`
	Line          []Line `xml:"line"                         json:"line"`
	Offset        *int64 `xml:"offset,attr,omitempty"        json:"offset,omitempty"`
	Synced        bool   `xml:"synced,attr"                  json:"synced"`
}

type StructuredLyrics []StructuredLyric
type LyricsList struct {
	StructuredLyrics []StructuredLyric `xml:"structuredLyrics,omitempty" json:"structuredLyrics,omitempty"`
}

type OpenSubsonicExtension struct {
	Name     string  `xml:"name,attr" json:"name"`
	Versions []int32 `xml:"versions"  json:"versions"`
}

type OpenSubsonicExtensions []OpenSubsonicExtension

type ItemGenre struct {
	Name string `xml:"name,attr" json:"name"`
}

// ItemGenres holds a list of genres (OpenSubsonic). If it is null, it must be marshalled as an empty array.
type ItemGenres []ItemGenre

func (i ItemGenres) MarshalJSON() ([]byte, error) {
	return marshalJSONArray(i)
}

type ReplayGain struct {
	TrackGain    float64 `xml:"trackGain,omitempty,attr"    json:"trackGain,omitempty"`
	AlbumGain    float64 `xml:"albumGain,omitempty,attr"    json:"albumGain,omitempty"`
	TrackPeak    float64 `xml:"trackPeak,omitempty,attr"    json:"trackPeak,omitempty"`
	AlbumPeak    float64 `xml:"albumPeak,omitempty,attr"    json:"albumPeak,omitempty"`
	BaseGain     float64 `xml:"baseGain,omitempty,attr"     json:"baseGain,omitempty"`
	FallbackGain float64 `xml:"fallbackGain,omitempty,attr" json:"fallbackGain,omitempty"`
}

type DiscTitle struct {
	Disc  int    `xml:"disc,attr,omitempty" json:"disc,omitempty"`
	Title string `xml:"title,attr,omitempty" json:"title,omitempty"`
}

type DiscTitles []DiscTitle

func (d DiscTitles) MarshalJSON() ([]byte, error) {
	return marshalJSONArray(d)
}

// marshalJSONArray marshals a slice of any type to JSON. If the slice is empty, it is marshalled as an
// empty array instead of null.
func marshalJSONArray[T any](v []T) ([]byte, error) {
	if len(v) == 0 {
		return json.Marshal([]T{})
	}
	a := v
	return json.Marshal(a)
}

type ItemDate struct {
	Year  int `xml:"year,attr,omitempty" json:"year,omitempty"`
	Month int `xml:"month,attr,omitempty" json:"month,omitempty"`
	Day   int `xml:"day,attr,omitempty" json:"day,omitempty"`
}
