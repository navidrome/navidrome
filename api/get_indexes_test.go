package api_test

import (
	"testing"

	"encoding/xml"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/consts"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/tests"
	"github.com/deluan/gosonic/tests/mocks"
	"github.com/deluan/gosonic/utils"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	emptyResponse = `<indexes lastModified="1" ignoredArticles="The El La Los Las Le Les Os As O A"></indexes>`
)

func TestGetIndexes(t *testing.T) {
	tests.Init(t, false)
	mockRepo := mocks.CreateMockArtistIndexRepo()
	utils.DefineSingleton(new(domain.ArtistIndexRepository), func() domain.ArtistIndexRepository {
		return mockRepo
	})
	propRepo := mocks.CreateMockPropertyRepo()
	utils.DefineSingleton(new(domain.PropertyRepository), func() domain.PropertyRepository {
		return propRepo
	})

	mockRepo.SetData("[]", 0)
	mockRepo.SetError(false)
	propRepo.Put(consts.LastScan, "1")
	propRepo.SetError(false)

	Convey("Subject: GetIndexes Endpoint", t, func() {
		Convey("Return fail on Index Table error", func() {
			mockRepo.SetError(true)
			_, w := Get(AddParams("/rest/getIndexes.view", "ifModifiedSince=0"), "TestGetIndexes")

			v := responses.Subsonic{}
			xml.Unmarshal(w.Body.Bytes(), &v)
			So(v.Status, ShouldEqual, "fail")
		})
		Convey("Return fail on Property Table error", func() {
			propRepo.SetError(true)
			_, w := Get(AddParams("/rest/getIndexes.view"), "TestGetIndexes")

			v := responses.Subsonic{}
			xml.Unmarshal(w.Body.Bytes(), &v)
			So(v.Status, ShouldEqual, "fail")
		})
		Convey("When the index is empty", func() {
			_, w := Get(AddParams("/rest/getIndexes.view"), "TestGetIndexes")

			Convey("Status code should be 200", func() {
				So(w.Code, ShouldEqual, 200)
			})
			Convey("It should return valid XML", func() {
				v := new(string)
				err := xml.Unmarshal(w.Body.Bytes(), &v)
				So(err, ShouldBeNil)
			})
			Convey("Then it should return an empty collection", func() {
				So(w.Body.String(), ShouldContainSubstring, emptyResponse)
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
			propRepo.Put(consts.LastScan, "1")

			_, w := Get(AddParams("/rest/getIndexes.view", "ifModifiedSince=2"), "TestGetIndexes")

			So(w.Body.String(), ShouldContainSubstring, emptyResponse)
		})
		Convey("And it should return empty if 'ifModifiedSince' is the asme as tie index last update", func() {
			mockRepo.SetData(`[{"Id": "A","Artists": [
				{"ArtistId": "21", "Artist": "Afrolicious"}
			]}]`, 2)
			propRepo.Put(consts.LastScan, "1")

			_, w := Get(AddParams("/rest/getIndexes.view", "ifModifiedSince=1"), "TestGetIndexes")

			So(w.Body.String(), ShouldContainSubstring, emptyResponse)
		})
		Reset(func() {
			mockRepo.SetData("[]", 0)
			mockRepo.SetError(false)
			propRepo.Put(consts.LastScan, "1")
			propRepo.SetError(false)
		})
	})
}
