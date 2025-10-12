//go:build go1.21

package dlna

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/anacrolix/dms/dlna"
	"github.com/anacrolix/dms/upnp"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/dlna/upnpav"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/oriser/regroup"
)

type contentDirectoryService struct {
	*DLNAServer
	upnp.Eventing
}

func (cds *contentDirectoryService) updateIDString() string {
	return fmt.Sprintf("%d", uint32(os.Getpid()))
}

// Turns the given entry and DMS host into a UPnP object. A nil object is
// returned if the entry is not of interest.
func (cds *contentDirectoryService) cdsObjectToUpnpavObject(cdsObject object, isContainer bool, host string) (ret interface{}) {
	obj := upnpav.Object{
		ID:         cdsObject.ID(),
		Restricted: 1,
		ParentID:   cdsObject.ParentID(),
		Title:      filepath.Base(cdsObject.Path),
	}

	if isContainer {
		defaultChildCount := 1
		obj.Class = "object.container.storageFolder"
		return upnpav.Container{
			Object:     obj,
			ChildCount: &defaultChildCount,
		}
	}
	// Read the mime type from the fs.Object if possible,
	// otherwise fall back to working out what it is from the file path.
	var mimeType = "audio/mp3" //TODO

	obj.Class = "object.item.audioItem"
	obj.Date = upnpav.Timestamp{Time: time.Now()}

	item := upnpav.Item{
		Object: obj,
		Res:    make([]upnpav.Resource, 0, 1),
	}

	item.Res = append(item.Res, upnpav.Resource{
		URL: (&url.URL{
			Scheme: "http",
			Host:   host,
			Path:   path.Join(resPath, cdsObject.Path),
		}).String(),
		ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", mimeType, dlna.ContentFeatures{
			SupportRange: true,
		}.String()),
		Size: uint64(1048576), //TODO
	})

	ret = item
	return ret
}

// Returns all the upnpav objects in a directory.
func (cds *contentDirectoryService) readContainer(o object, host string) (ret []interface{}, err error) {
	log.Debug(fmt.Sprintf("ReadContainer called '%s'", o))

	if o.Path == "/" || o.Path == "" {
		log.Debug("ReadContainer default route")
		newObject := object{Path: "/Music"}
		ret = append(ret, cds.cdsObjectToUpnpavObject(newObject, true, host))
		return ret, nil
	}

	filesRegex := regroup.MustCompile("\\/Music\\/Files[\\/]?((?P<Path>.+))?")
	artistRegex := regroup.MustCompile("\\/Music\\/Artists[\\/]?(?P<Artist>[^\\/]+)?[\\/]?(?<ArtistAlbum>[^\\/]+)?[\\/]?(?<ArtistAlbumTrack>[^\\/]+)?")
	albumRegex := regroup.MustCompile("\\/Music\\/Albums[\\/]?(?P<AlbumTitle>[^\\/]+)?[\\/]?(?<AlbumTrack>[^\\/]+)?")
	genresRegex := regroup.MustCompile("\\/Music\\/Genres[\\/]?(?P<Genre>[^\\/]+)?[\\/]?(?P<GenreArtist>[^/]+)?[\\/]?(?P<GenreTrack>[^\\/]+)?")
	recentRegex := regroup.MustCompile("\\/Music\\/Recently Added[\\/]?(?P<RecentTrack>[^\\/]+)?")
	playlistRegex := regroup.MustCompile("\\/Music\\/Playlist[\\/]?(?P<Playlist>[^\\/]+)?[\\/]?(?P<PlaylistTrack>[^\\/]+)?")

	if o.Path == "/Music" {
		ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Files"}, true, host))
		ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Artists"}, true, host))
		ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Albums"}, true, host))
		ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Genres"}, true, host))
		ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Recently Added"}, true, host))
		ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Playlists"}, true, host))
		return ret, nil
	} else if _, err := filesRegex.Groups(o.Path); err == nil {
		return cds.doFiles(ret, o.Path, host)
	} else if matchResults, err := artistRegex.Groups(o.Path); err == nil {
		log.Debug(fmt.Sprintf("Artist MATCH: %+v", matchResults))
		if matchResults["ArtistAlbumTrack"] != "" {
			//TODO
			log.Debug("Artist Get a track ")
		} else if matchResults["ArtistAlbum"] != "" {
			log.Debug("Artist Get an album ")
			album := matchResults["ArtistAlbum"]

			albumResponse, _ := cds.ds.Album(cds.ctx).Get(album)
			log.Debug(fmt.Sprintf("Album Returned: %+v for %s", albumResponse, album))
			basePath := path.Join("/Music/Artists", matchResults["Artist"], matchResults["ArtistAlbum"])
			return cds.doAlbum(albumResponse, basePath, ret, host)

		} else if matchResults["Artist"] != "" {
			log.Debug(fmt.Sprintf("Artist Get an Artist: %s", matchResults["Artist"]))
			allAlbumsForThisArtist, _ := cds.ds.Album(cds.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_artist_id": matchResults["Artist"]}})
			basePath := path.Join("/Music/Artists", matchResults["Artist"])
			return cds.doAlbums(allAlbumsForThisArtist, basePath, ret, host)

		} else {
			indexes, err := cds.ds.Artist(cds.ctx).GetIndex()
			if err != nil {
				fmt.Printf("Error retrieving Indexes: %+v", err)
				return nil, err
			}
			for letterIndex := range indexes {
				for artist := range indexes[letterIndex].Artists {
					artistId := indexes[letterIndex].Artists[artist].ID
					child := object{
						Path: path.Join(o.Path, indexes[letterIndex].Artists[artist].Name),
						Id:   path.Join(o.Path, artistId),
					}
					ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
				}
			}
			return ret, nil
		}
	} else if matchResults, err := albumRegex.Groups(o.Path); err == nil {
		log.Debug("Album MATCH")
		if matchResults["AlbumTrack"] != "" {
			log.Debug("AlbumTrack MATCH")
			//TODO
		} else if matchResults["AlbumTitle"] != "" {
			log.Debug("AlbumTitle MATCH")
			x, _ := cds.ds.Album(cds.ctx).Get(matchResults["AlbumTitle"])
			basePath := "/Music/Albums"
			return cds.doAlbum(x, basePath, ret, host)
		} else {
			log.Debug("albumRegex else MATCH")
			indexes, err := cds.ds.Album(cds.ctx).GetAllWithoutGenres()
			if err != nil {
				fmt.Printf("Error retrieving Indexes: %+v", err)
				return nil, err
			}
			for indexItem := range indexes {
				child := object{
					Path: path.Join(o.Path, indexes[indexItem].Name),
					Id:   path.Join(o.Path, indexes[indexItem].ID),
				}
				ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
			}
			return ret, nil
		}
	} else if matchResults, err := genresRegex.Groups(o.Path); err == nil {
		log.Debug("Genre MATCH")
		if _, exists := matchResults["GenreTrack"]; exists {
			log.Debug("GenreTrack MATCH")
			//TODO
		} else if _, exists := matchResults["GenreArtist"]; exists {
			log.Debug("GenreArtist MATCH")
			//TODO
		} else if genre, exists := matchResults["Genre"]; exists {
			log.Debug("Genre only MATCH")
			x, xerr := cds.ds.Album(cds.ctx).Get(genre)
			log.Debug(fmt.Sprintf("Genre: %+v", x), xerr)
		} else {
			log.Debug("Genre else MATCH")
			indexes, err := cds.ds.Genre(cds.ctx).GetAll()
			if err != nil {
				fmt.Printf("Error retrieving Indexes: %+v", err)
				return nil, err
			}
			for indexItem := range indexes {
				child := object{
					Path: path.Join(o.Path, indexes[indexItem].Name),
					Id:   path.Join(o.Path, indexes[indexItem].ID),
				}
				ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
			}
			return ret, nil
		}
	} else if matchResults, err := recentRegex.Groups(o.Path); err == nil {
		log.Debug("recent MATCH")
		fmt.Printf("%+v",matchResults)
	} else if matchResults, err := playlistRegex.Groups(o.Path); err == nil {
		log.Debug("Playlist MATCH")
		if _, exists := matchResults["PlaylistTrack"]; exists {
			log.Debug("PlaylistTrack MATCH")
		} else if playlist, exists := matchResults["Playlist"]; exists {
			log.Debug("Playlist only MATCH")
			x, xerr := cds.ds.Playlist(cds.ctx).Get(playlist)
				log.Debug(fmt.Sprintf("Playlist: %+v", x), xerr)
		} else {
			log.Debug("Playlist else MATCH")
			indexes, err := cds.ds.Playlist(cds.ctx).GetAll()
			if err != nil {
				fmt.Printf("Error retrieving Indexes: %+v", err)
				return nil, err
			}
			for indexItem := range indexes {
				child := object{
					Path: path.Join(o.Path, indexes[indexItem].Name),
					Id:   path.Join(o.Path, indexes[indexItem].ID),
				}
				ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
			}
			return ret, nil
		}
	}
		/*
		   		deluan
		    —
		   Today at 18:30
		   ds.Album(ctx).GetAll(FIlter: Eq{"albumArtistId": artistID})
		   Or something like that 😛
		   Mintsoft
		    —
		   Today at 18:30
		   For other examples, how do I know what the right magic string for "albumArtistId" is?
		   kgarner7
		    —
		   Today at 18:31
		   album_artist_id
		   Look at the model structs names
		   deluan
		    —
		   Today at 18:31
		   This is a limitation of Squirrel. It is string based. YOu have to use the name of the columns in the DB
		*/
		return
}

func (cds *contentDirectoryService) doAlbum(album *model.Album, basepath string, ret []interface{}, host string) ([]interface{}, error) {
	log.Debug(fmt.Sprintf("TODO: doAlbum Called with : '%+v', '%s'", album, basepath))
	return ret, nil
}

func (cds *contentDirectoryService) doAlbums(albums model.Albums, basepath string, ret []interface{}, host string) ([]interface{}, error) {
	for _, album := range albums {
		child := object {
			Path: path.Join(basepath, album.Name),
			Id: path.Join(basepath, album.ID),
		}
		ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
	}
	return ret, nil
}

func (cds *contentDirectoryService) doFiles(ret []interface{}, oPath string, host string) ([]interface{}, error) {
	pathComponents := strings.Split(strings.TrimPrefix(oPath, "/Music/Files"), "/")
	if slices.Contains(pathComponents, "..") || slices.Contains(pathComponents, ".") {
		log.Error("Attempt to use .. or . detected", oPath, host)
		return ret, nil
	}
	totalPathArrayBits := append([]string{conf.Server.MusicFolder}, pathComponents...)
	localFilePath := filepath.Join(totalPathArrayBits...)

	files, _ := os.ReadDir(localFilePath)
	for _, file := range files {
		child := object{
			Path: path.Join(oPath, file.Name()),
			Id:   path.Join(oPath, file.Name()),
		}
		ret = append(ret, cds.cdsObjectToUpnpavObject(child, file.IsDir(), host))
	}
	return ret, nil
}

type browse struct {
	ObjectID       string
	BrowseFlag     string
	Filter         string
	StartingIndex  int
	RequestedCount int
}

// ContentDirectory object from ObjectID.
func (cds *contentDirectoryService) objectFromID(id string) (o object, err error) {
	log.Debug("objectFromID called", "id", id)

	o.Path, err = url.QueryUnescape(id)
	if err != nil {
		return
	}
	if o.Path == "0" {
		o.Path = "/"
	}
	o.Path = path.Clean(o.Path)
	if !path.IsAbs(o.Path) {
		err = fmt.Errorf("bad ObjectID %v", o.Path)
		return
	}
	return
}

func (cds *contentDirectoryService) Handle(action string, argsXML []byte, r *http.Request) (map[string]string, error) {
	host := r.Host
	log.Info(fmt.Sprintf("Handle called with action: %s", action))

	switch action {
	case "GetSystemUpdateID":
		return map[string]string{
			"Id": cds.updateIDString(),
		}, nil
	case "GetSortCapabilities":
		return map[string]string{
			"SortCaps": "dc:title",
		}, nil
	case "Browse":
		var browse browse
		if err := xml.Unmarshal(argsXML, &browse); err != nil {
			return nil, err
		}
		obj, err := cds.objectFromID(browse.ObjectID)
		if err != nil {
			return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, "%s", err.Error())
		}
		switch browse.BrowseFlag {
		case "BrowseDirectChildren":
			objs, err := cds.readContainer(obj, host)
			if err != nil {
				return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, "%s", err.Error())
			}
			totalMatches := len(objs)
			objs = objs[func() (low int) {
				low = browse.StartingIndex
				if low > len(objs) {
					low = len(objs)
				}
				return
			}():]
			if browse.RequestedCount != 0 && browse.RequestedCount < len(objs) {
				objs = objs[:browse.RequestedCount]
			}
			result, err := xml.Marshal(objs)
			if err != nil {
				return nil, err
			}
			return map[string]string{
				"TotalMatches":   fmt.Sprint(totalMatches),
				"NumberReturned": fmt.Sprint(len(objs)),
				"Result":         didlLite(string(result)),
				"UpdateID":       cds.updateIDString(),
			}, nil
		case "BrowseMetadata":
			//TODO
			return map[string]string{
				"Result": didlLite(string("result")),
			}, nil
		default:
			return nil, upnp.Errorf(upnp.ArgumentValueInvalidErrorCode, "unhandled browse flag: %v", browse.BrowseFlag)
		}
	case "GetSearchCapabilities":
		return map[string]string{
			"SearchCaps": "",
		}, nil
	// Samsung Extensions
	case "X_GetFeatureList":
		return map[string]string{
			"FeatureList": `<Features xmlns="urn:schemas-upnp-org:av:avs" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:schemas-upnp-org:av:avs http://www.upnp.org/schemas/av/avs.xsd">
	<Feature name="samsung.com_BASICVIEW" version="1">
		<container id="0" type="object.item.imageItem"/>
		<container id="0" type="object.item.audioItem"/>
		<container id="0" type="object.item.videoItem"/>
	</Feature>
</Features>`}, nil
	case "X_SetBookmark":
		// just ignore
		return map[string]string{}, nil
	default:
		return nil, upnp.InvalidActionError
	}
}

// Represents a ContentDirectory object.
type object struct {
	Path string // The cleaned, absolute path for the object relative to the server.
	Id   string
}

// Returns the actual local filesystem path for the object.
func (o *object) FilePath() string {
	return filepath.FromSlash(o.Path)
}

// Returns the ObjectID for the object. This is used in various ContentDirectory actions.
func (o object) ID() string {
	if o.Id != "" {
		return o.Id
	}
	if !path.IsAbs(o.Path) {
		log.Fatal(fmt.Sprintf("Relative object path used with ID: $s", o.Path))
	}
	if len(o.Path) == 1 {
		return "0"
	}
	return url.QueryEscape(o.Path)
}

func (o *object) IsRoot() bool {
	return o.Path == "/"
}

// Returns the object's parent ObjectID. Fortunately it can be deduced from the
// ObjectID (for now).
func (o object) ParentID() string {
	if o.IsRoot() {
		return "-1"
	}
	o.Path = path.Dir(o.Path)
	return o.ID()
}
