package persistence

import (
	"strconv"
	"testing"

	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIndexRepository(t *testing.T) {

	tests.Init(t, false)

	Convey("Subject: NewIndexRepository", t, func() {
		repo := NewArtistIndexRepository()

		Convey("It should be able to read and write to the database", func() {
			i := &domain.ArtistIndex{Id: "123"}

			repo.Put(i)
			s, _ := repo.Get("123")

			So(s, shouldBeEqual, i)
		})
		Convey("It should be able to check for existance of an Id", func() {
			i := &domain.ArtistIndex{Id: "123"}

			repo.Put(i)

			s, _ := repo.Exists("123")
			So(s, ShouldBeTrue)

			s, _ = repo.Exists("NOT_FOUND")
			So(s, ShouldBeFalse)
		})
		Convey("Method Put() should return error if Id is not set", func() {
			i := &domain.ArtistIndex{}

			err := repo.Put(i)

			So(err, ShouldNotBeNil)
		})
		Convey("Given that I have 4 records", func() {
			for i := 1; i <= 4; i++ {
				e := &domain.ArtistIndex{Id: strconv.Itoa(i)}
				repo.Put(e)
			}

			Convey("When I call GetAll()", func() {
				indices, err := repo.GetAll()
				Convey("Then It should not return any error", func() {
					So(err, ShouldBeNil)
				})
				Convey("And It should return 4 entities", func() {
					So(indices, ShouldHaveLength, 4)
				})
				Convey("And the values should be retrieved", func() {
					for _, e := range indices {
						So(e.Id, ShouldBeIn, []string{"1", "2", "3", "4"})
					}
				})
			})
		})
		Reset(func() {
			dropDb()
		})
	})
}
