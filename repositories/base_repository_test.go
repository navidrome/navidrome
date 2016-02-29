package repositories

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/deluan/gosonic/tests"
)

func TestUnitBaseRepository(t *testing.T) {
	tests.Init(t, false)

	Convey("Subject: NewId", t, func() {

		repo := &BaseRepository{table: "test_table"}

		Convey("When I call NewId with a name", func() {
			Id := repo.NewId("a name")
			Convey("Then it should return a new Id", func() {
				So(Id, ShouldNotBeEmpty)
			})
		})

		Convey("When I call NewId with the same name twice", func() {
			FirstId := repo.NewId("a name")
			SecondId := repo.NewId("a name")

			Convey("Then it should return the same Id each time", func() {
				So(FirstId, ShouldEqual, SecondId)
			})

		})

		Convey("When I call NewId with different names", func() {
			FirstId := repo.NewId("first name")
			SecondId := repo.NewId("second name")

			Convey("Then it should return different Ids", func() {
				So(FirstId, ShouldNotEqual, SecondId)
			})

		})

	})
}
