package api_test

import (
	"testing"

	"github.com/deluan/gosonic/utils"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/deluan/gosonic/tests"
	"github.com/deluan/gosonic/models"
	"github.com/deluan/gosonic/repositories"
	"encoding/xml"
	"encoding/json"
	"fmt"
	"errors"
	"github.com/deluan/gosonic/api/responses"
)

func TestGetIndexes(t *testing.T) {
	tests.Init(t, false)
	mockRepo := &mockArtistIndex{}
	utils.DefineSingleton(new(repositories.ArtistIndex), func() repositories.ArtistIndex {
		return mockRepo
	})

	Convey("Subject: GetIndexes Endpoint", t, func() {
		Convey("Return fail on DB error", func() {
			mockRepo.err = true
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
				So(w.Body.String(), ShouldContainSubstring, "<indexes></indexes>")
			})
		})
		Convey("When the index is not empty", func() {
			mockRepo.data = makeMockData(`[{"Id": "A","Artists": []}]`, 2)
			_, w := Get(AddParams("/rest/getIndexes.view"), "TestGetIndexes")

			Convey("Then it should return the the items in the response", func() {
				So(w.Body.String(), ShouldContainSubstring, `<index name="A"></index>`)
			})
		})
		Reset(func() {
			mockRepo.data = make([]models.ArtistIndex, 0)
			mockRepo.err = false
		})
	})
}

func makeMockData(j string, length int) []models.ArtistIndex {
	data := make([]models.ArtistIndex, length)
	err := json.Unmarshal([]byte(j), &data)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	return data
}

type mockArtistIndex struct {
	repositories.ArtistIndexImpl
	data []models.ArtistIndex
	err  bool
}

func (m *mockArtistIndex) GetAll() ([]models.ArtistIndex, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.data, nil
}