package responses_test

import (
	. "github.com/deluan/gosonic/api/responses"
	. "github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestSubsonicResponses(t *testing.T) {

	response := &Subsonic{Status: "ok", Version: "1.0.0"}

	Convey("Subject: Subsonic Responses", t, func() {
		Convey("EmptyResponse", func() {
			Convey("XML", func() {
				So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"></subsonic-response>`)
			})
			Convey("JSON", func() {
				So(response, ShouldMatchJSON, `{"status":"ok","version":"1.0.0"}`)
			})
		})

		Convey("License", func() {
			response.License = &License{Valid: true}
			Convey("XML", func() {
				So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"><license valid="true"></license></subsonic-response>`)
			})
			Convey("JSON", func() {
				So(response, ShouldMatchJSON, `{"license":{"valid":true},"status":"ok","version":"1.0.0"}`)
			})
		})

		Convey("MusicFolders", func() {
			response.MusicFolders = &MusicFolders{}

			Convey("With data", func() {
				folders := make([]MusicFolder, 2)
				folders[0] = MusicFolder{Id: "111", Name: "aaa"}
				folders[1] = MusicFolder{Id: "222", Name: "bbb"}
				response.MusicFolders.Folders = folders

				Convey("XML", func() {
					So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"><musicFolders><musicFolder id="111" name="aaa"></musicFolder><musicFolder id="222" name="bbb"></musicFolder></musicFolders></subsonic-response>`)
				})
				Convey("JSON", func() {
					So(response, ShouldMatchJSON, `{"musicFolders":{"musicFolder":[{"id":"111","name":"aaa"},{"id":"222","name":"bbb"}]},"status":"ok","version":"1.0.0"}`)
				})
			})
			Convey("Without data", func() {
				Convey("XML", func() {
					So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"><musicFolders></musicFolders></subsonic-response>`)
				})
				Convey("JSON", func() {
					So(response, ShouldMatchJSON, `{"musicFolders":{},"status":"ok","version":"1.0.0"}`)
				})
			})
		})

		Convey("Indexes", func() {
			artists := make([]Artist, 1)
			artists[0] = Artist{Id: "111", Name: "aaa"}
			response.Indexes = &Indexes{LastModified: "1", IgnoredArticles: "A"}

			Convey("With data", func() {
				index := make([]Index, 1)
				index[0] = Index{Name: "A", Artists: artists}
				response.Indexes.Index = index
				Convey("XML", func() {
					So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"><indexes lastModified="1" ignoredArticles="A"><index name="A"><artist id="111" name="aaa"></artist></index></indexes></subsonic-response>`)
				})
				Convey("JSON", func() {
					So(response, ShouldMatchJSON, `{"indexes":{"ignoredArticles":"A","index":[{"artist":[{"id":"111","name":"aaa"}],"name":"A"}],"lastModified":"1"},"status":"ok","version":"1.0.0"}`)
				})
			})
			Convey("Without data", func() {
				Convey("XML", func() {
					So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"><indexes lastModified="1" ignoredArticles="A"></indexes></subsonic-response>`)
				})
				Convey("JSON", func() {
					So(response, ShouldMatchJSON, `{"indexes":{"ignoredArticles":"A","lastModified":"1"},"status":"ok","version":"1.0.0"}`)
				})
			})
		})

		Convey("Directory", func() {
			response.Directory = &Directory{Id: "1", Name: "N"}
			Convey("Without data", func() {
				Convey("XML", func() {
					So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"><directory id="1" name="N"></directory></subsonic-response>`)
				})
				Convey("JSON", func() {
					So(response, ShouldMatchJSON, `{"directory":{"id":"1","name":"N"},"status":"ok","version":"1.0.0"}`)
				})
			})
			Convey("With just required data", func() {
				child := make([]Child, 1)
				child[0] = Child{Id: "1", Title: "title", IsDir: false}
				response.Directory.Child = child
				Convey("XML", func() {
					So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"><directory id="1" name="N"><child id="1" isDir="false" title="title"></child></directory></subsonic-response>`)
				})
				Convey("JSON", func() {
					So(response, ShouldMatchJSON, `{"directory":{"child":[{"id":"1","isDir":false,"title":"title"}],"id":"1","name":"N"},"status":"ok","version":"1.0.0"}`)
				})
			})
			Convey("With all data", func() {
				child := make([]Child, 1)
				t := time.Date(2016, 03, 2, 20, 30, 0, 0, time.UTC)
				child[0] = Child{
					Id: "1", IsDir: true, Title: "title", Album: "album", Artist: "artist", Track: 1,
					Year: 1985, Genre: "Rock", CoverArt: "1", Size: "8421341", ContentType: "audio/flac",
					Suffix: "flac", TranscodedContentType: "audio/mpeg", TranscodedSuffix: "mp3",
					Duration: 146, BitRate: 320, Starred: &t,
				}
				response.Directory.Child = child
				Convey("XML", func() {
					So(response, ShouldMatchXML, `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.0.0"><directory id="1" name="N"><child id="1" isDir="true" title="title" album="album" artist="artist" track="1" year="1985" genre="Rock" coverArt="1" size="8421341" contentType="audio/flac" suffix="flac" starred="2016-03-02T20:30:00Z" transcodedContentType="audio/mpeg" transcodedSuffix="mp3" duration="146" bitRate="320"></child></directory></subsonic-response>`)
				})
				Convey("JSON", func() {
					So(response, ShouldMatchJSON, `{"directory":{"child":[{"album":"album","artist":"artist","bitRate":320,"contentType":"audio/flac","coverArt":"1","duration":146,"genre":"Rock","id":"1","isDir":true,"size":"8421341","starred":"2016-03-02T20:30:00Z","suffix":"flac","title":"title","track":1,"transcodedContentType":"audio/mpeg","transcodedSuffix":"mp3","year":1985}],"id":"1","name":"N"},"status":"ok","version":"1.0.0"}`)
				})
			})
		})

		Reset(func() {
			response = &Subsonic{Status: "ok", Version: "1.0.0"}
		})
	})

}
