package responses

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	. "github.com/deluan/gosonic/tests"
)

func TestSubsonicResponses(t *testing.T) {

	response := &Subsonic{Status: "ok", Version: "1.0.0"}

	Convey("Subject: Subsonic Responses", t, func(){
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
			response.Indexes = &Indexes{LastModified:"1", IgnoredArticles:"A"}

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

		Reset(func() {
			response = &Subsonic{Status: "ok", Version: "1.0.0"}
		})
	})

}