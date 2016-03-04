package persistence

import (
	"fmt"
	"github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"strconv"
	"testing"
)

type TestEntity struct {
	Id   string
	Name string
}

func shouldBeEqual(actualStruct interface{}, expectedStruct ...interface{}) string {
	actual := fmt.Sprintf("%#v", actualStruct)
	expected := fmt.Sprintf("%#v", expectedStruct[0])
	return ShouldEqual(actual, expected)
}

func createRepo() *ledisRepository {
	repo := &ledisRepository{}
	repo.init("test", &TestEntity{})
	return repo
}

func TestBaseRepository(t *testing.T) {
	tests.Init(t, false)

	Convey("Subject: NewId", t, func() {
		repo := createRepo()

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

	Convey("Subject: saveOrUpdate/loadEntity/CountAll", t, func() {

		Convey("Given an empty DB", func() {
			repo := createRepo()

			Convey("When I save a new entity", func() {
				entity := &TestEntity{"123", "My Name"}
				err := repo.saveOrUpdate("123", entity)

				Convey("Then the method shouldn't return any errors", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the number of entities should be 1", func() {
					count, _ := repo.CountAll()
					So(count, ShouldEqual, 1)
				})

				Convey("And this entity should be equal to the the saved one", func() {
					actualEntity, _ := repo.readEntity("123")
					So(actualEntity, shouldBeEqual, entity)
				})

			})

		})

		Convey("Given a table with one entity", func() {
			repo := createRepo()
			entity := &TestEntity{"111", "One Name"}
			repo.saveOrUpdate(entity.Id, entity)

			Convey("When I save an entity with a different Id", func() {
				newEntity := &TestEntity{"222", "Another Name"}
				repo.saveOrUpdate(newEntity.Id, newEntity)

				Convey("Then the number of entities should be 2", func() {
					count, _ := repo.CountAll()
					So(count, ShouldEqual, 2)
				})

			})

			Convey("When I save an entity with the same Id", func() {
				newEntity := &TestEntity{"111", "New Name"}
				repo.saveOrUpdate(newEntity.Id, newEntity)

				Convey("Then the number of entities should be 1", func() {
					count, _ := repo.CountAll()
					So(count, ShouldEqual, 1)
				})

				Convey("And the entity should be updated", func() {
					e, _ := repo.readEntity("111")
					actualEntity := e.(*TestEntity)
					So(actualEntity.Name, ShouldEqual, newEntity.Name)
				})

			})

		})

		Convey("Given a table with 3 entities", func() {
			repo := createRepo()
			for i := 1; i <= 3; i++ {
				e := &TestEntity{strconv.Itoa(i), fmt.Sprintf("Name %d", i)}
				repo.saveOrUpdate(e.Id, e)
			}

			Convey("When I call loadAll", func() {
				var es = make([]TestEntity, 0)
				err := repo.loadAll(&es)
				Convey("Then It should not return any error", func() {
					So(err, ShouldBeNil)
				})
				Convey("And I should get 3 entities", func() {
					So(len(es), ShouldEqual, 3)
				})
				Convey("And the values should be retrieved", func() {
					for _, e := range es {
						So(e.Id, ShouldBeIn, []string{"1", "2", "3"})
						So(e.Name, ShouldBeIn, []string{"Name 1", "Name 2", "Name 3"})
					}
				})
			})
		})

		Reset(func() {
			dropDb()
		})

	})
}
