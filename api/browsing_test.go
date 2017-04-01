package api_test

import (
	"testing"

	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/persistence"
	. "github.com/cloudsonic/sonic-server/tests"
	"github.com/cloudsonic/sonic-server/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMusicFolders(t *testing.T) {
	Init(t, false)

	_, w := Get(AddParams("/rest/getMusicFolders.view"), "TestGetMusicFolders")

	Convey("Subject: GetMusicFolders Endpoint", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The response should include the default folder", func() {
			So(UnindentJSON(w.Body.Bytes()), ShouldContainSubstring, `{"musicFolder":[{"id":"0","name":"iTunes Library"}]}`)
		})
	})
}

const (
	emptyResponse = `{"indexes":{"ignoredArticles":"The El La Los Las Le Les Os As O A","lastModified":"1"}`
)

func TestGetIndexes(t *testing.T) {
	Init(t, false)

	mockRepo := persistence.CreateMockArtistIndexRepo()
	utils.DefineSingleton(new(domain.ArtistIndexRepository), func() domain.ArtistIndexRepository {
		return mockRepo
	})
	propRepo := engine.CreateMockPropertyRepo()
	utils.DefineSingleton(new(engine.PropertyRepository), func() engine.PropertyRepository {
		return propRepo
	})

	mockRepo.SetData("[]", 0)
	mockRepo.SetError(false)
	propRepo.Put(engine.PropLastScan, "1")
	propRepo.SetError(false)

	Convey("Subject: GetIndexes Endpoint", t, func() {
		Convey("Return fail on Index Table error", func() {
			mockRepo.SetError(true)
			_, w := Get(AddParams("/rest/getIndexes.view", "ifModifiedSince=0"), "TestGetIndexes")

			So(w.Body, ShouldReceiveError, responses.ErrorGeneric)
		})
		Convey("Return fail on Property Table error", func() {
			propRepo.SetError(true)
			_, w := Get(AddParams("/rest/getIndexes.view"), "TestGetIndexes")

			So(w.Body, ShouldReceiveError, responses.ErrorGeneric)
		})
		Convey("When the index is empty", func() {
			_, w := Get(AddParams("/rest/getIndexes.view"), "TestGetIndexes")

			Convey("Status code should be 200", func() {
				So(w.Code, ShouldEqual, 200)
			})
			Convey("Then it should return an empty collection", func() {
				So(UnindentJSON(w.Body.Bytes()), ShouldContainSubstring, emptyResponse)
			})
		})
		Convey("When the index is not empty", func() {
			mockRepo.SetData(`[{"Id": "A","Artists": [
				{"ArtistId": "21", "Artist": "Afrolicious"}
			]}]`, 2)

			SkipConvey("Then it should return the the items in the response", func() {
				_, w := Get(AddParams("/rest/getIndexes.view"), "TestGetIndexes")

				So(w.Body.String(), ShouldContainSubstring,
					`<index name="A"><artist id="21" name="Afrolicious"></artist></index>`)
			})
		})
		Convey("And it should return empty if 'ifModifiedSince' is more recent than the index", func() {
			mockRepo.SetData(`[{"Id": "A","Artists": [
				{"ArtistId": "21", "Artist": "Afrolicious"}
			]}]`, 2)
			propRepo.Put(engine.PropLastScan, "1")

			_, w := Get(AddParams("/rest/getIndexes.view", "ifModifiedSince=2"), "TestGetIndexes")

			So(UnindentJSON(w.Body.Bytes()), ShouldContainSubstring, emptyResponse)
		})
		Convey("And it should return empty if 'ifModifiedSince' is the asme as tie index last update", func() {
			mockRepo.SetData(`[{"Id": "A","Artists": [
				{"ArtistId": "21", "Artist": "Afrolicious"}
			]}]`, 2)
			propRepo.Put(engine.PropLastScan, "1")

			_, w := Get(AddParams("/rest/getIndexes.view", "ifModifiedSince=1"), "TestGetIndexes")

			So(UnindentJSON(w.Body.Bytes()), ShouldContainSubstring, emptyResponse)
		})
		Reset(func() {
			mockRepo.SetData("[]", 0)
			mockRepo.SetError(false)
			propRepo.Put(engine.PropLastScan, "1")
			propRepo.SetError(false)
		})
	})
}

func TestGetMusicDirectory(t *testing.T) {
	Init(t, false)

	mockArtistRepo := persistence.CreateMockArtistRepo()
	utils.DefineSingleton(new(domain.ArtistRepository), func() domain.ArtistRepository {
		return mockArtistRepo
	})
	mockAlbumRepo := persistence.CreateMockAlbumRepo()
	utils.DefineSingleton(new(domain.AlbumRepository), func() domain.AlbumRepository {
		return mockAlbumRepo
	})
	mockMediaFileRepo := persistence.CreateMockMediaFileRepo()
	utils.DefineSingleton(new(domain.MediaFileRepository), func() domain.MediaFileRepository {
		return mockMediaFileRepo
	})

	Convey("Subject: GetMusicDirectory Endpoint", t, func() {
		Convey("Should fail if missing Id parameter", func() {
			_, w := Get(AddParams("/rest/getMusicDirectory.view"), "TestGetMusicDirectory")

			So(w.Body, ShouldReceiveError, responses.ErrorMissingParameter)
		})
		Convey("Id is for an artist", func() {
			Convey("Return fail on Artist Table error", func() {
				mockArtistRepo.SetData(`[{"Id":"1","Name":"The Charlatans"}]`, 1)
				mockArtistRepo.SetError(true)
				_, w := Get(AddParams("/rest/getMusicDirectory.view", "id=1"), "TestGetMusicDirectory")

				So(w.Body, ShouldReceiveError, responses.ErrorGeneric)
			})
		})
		Convey("When id is not found", func() {
			mockArtistRepo.SetData(`[{"Id":"1","Name":"The Charlatans"}]`, 1)
			_, w := Get(AddParams("/rest/getMusicDirectory.view", "id=NOT_FOUND"), "TestGetMusicDirectory")

			So(w.Body, ShouldReceiveError, responses.ErrorDataNotFound)
		})
		Convey("When id matches an artist", func() {
			mockArtistRepo.SetData(`[{"Id":"1","Name":"The KLF"}]`, 1)

			Convey("Without albums", func() {
				_, w := Get(AddParams("/rest/getMusicDirectory.view", "id=1"), "TestGetMusicDirectory")

				So(w.Body, ShouldContainJSON, `"id":"1","name":"The KLF"`)
			})
			Convey("With albums", func() {
				mockAlbumRepo.SetData(`[{"Id":"A","Name":"Tardis","ArtistId":"1"}]`, 1)
				_, w := Get(AddParams("/rest/getMusicDirectory.view", "id=1"), "TestGetMusicDirectory")

				So(w.Body, ShouldContainJSON, `"child":[{"album":"Tardis","albumId":"A","artistId":"1","id":"A","isDir":true,"parent":"1","title":"Tardis"}]`)
			})
		})
		Convey("When id matches an album with tracks", func() {
			mockArtistRepo.SetData(`[{"Id":"2","Name":"Céu"}]`, 1)
			mockAlbumRepo.SetData(`[{"Id":"A","Name":"Vagarosa","ArtistId":"2"}]`, 1)
			mockMediaFileRepo.SetData(`[{"Id":"3","Title":"Cangote","AlbumId":"A"}]`, 1)
			_, w := Get(AddParams("/rest/getMusicDirectory.view", "id=A"), "TestGetMusicDirectory")

			So(w.Body, ShouldContainJSON, `"child":[{"albumId":"A","id":"3","isDir":false,"parent":"A","title":"Cangote","type":"music"}]`)
		})
		Reset(func() {
			mockArtistRepo.SetData("[]", 0)
			mockArtistRepo.SetError(false)

			mockAlbumRepo.SetData("[]", 0)
			mockAlbumRepo.SetError(false)

			mockMediaFileRepo.SetData("[]", 0)
			mockMediaFileRepo.SetError(false)
		})
	})
}
