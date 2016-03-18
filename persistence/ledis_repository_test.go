package persistence

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/tests"
	"github.com/deluan/gosonic/utils"
	. "github.com/smartystreets/goconvey/convey"
)

type TestEntity struct {
	Id       string
	Name     string
	ParentId string    `parent:"parent"`
	Year     time.Time `idx:"ByYear"`
	Count    int       `idx:"ByCount"`
	Flag     bool      `idx:"ByFlag"`
}

func shouldBeEqual(actualStruct interface{}, expectedStruct ...interface{}) string {
	actual := fmt.Sprintf("%#v", actualStruct)
	expected := fmt.Sprintf("%#v", expectedStruct[0])
	return ShouldEqual(actual, expected)
}

func createEmptyRepo() *ledisRepository {
	dropDb()
	repo := &ledisRepository{}
	repo.init("test", &TestEntity{})
	return repo
}

func TestBaseRepository(t *testing.T) {
	tests.Init(t, false)

	Convey("Subject: Annotations", t, func() {
		repo := createEmptyRepo()
		Convey("It should parse the parent table definition", func() {
			So(repo.parentTable, ShouldEqual, "parent")
			So(repo.parentIdField, ShouldEqual, "ParentId")
		})
		Convey("It should parse the definded indexes", func() {
			So(repo.indexes, ShouldHaveLength, 3)
			So(repo.indexes["ByYear"], ShouldEqual, "Year")
			So(repo.indexes["ByFlag"], ShouldEqual, "Flag")
			So(repo.indexes["ByCount"], ShouldEqual, "Count")
		})
	})

	Convey("Subject: calcScore", t, func() {
		repo := createEmptyRepo()

		Convey("It should create an int score", func() {
			def := repo.indexes["ByCount"]
			entity := &TestEntity{Count: 10}
			score := calcScore(entity, def)

			So(score, ShouldEqual, 10)
		})
		Convey("It should create a boolean score", func() {
			def := repo.indexes["ByFlag"]
			Convey("Value false", func() {
				entity := &TestEntity{Flag: false}
				score := calcScore(entity, def)

				So(score, ShouldEqual, 0)
			})
			Convey("Value true", func() {
				entity := &TestEntity{Flag: true}
				score := calcScore(entity, def)

				So(score, ShouldEqual, 1)
			})
		})
		Convey("It should create a time score", func() {
			def := repo.indexes["ByYear"]
			now := time.Now()
			entity := &TestEntity{Year: now}
			score := calcScore(entity, def)

			So(score, ShouldEqual, utils.ToMillis(now))
		})
	})

	Convey("Subject: NewId", t, func() {
		repo := createEmptyRepo()

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
			repo := createEmptyRepo()

			Convey("When I try to retrieve an nonexistent ID", func() {
				_, err := repo.readEntity("NOT_FOUND")
				Convey("Then I should get a NotFound error", func() {
					So(err, ShouldEqual, domain.ErrNotFound)
				})
			})

			Convey("When I save a new entity and a parent", func() {
				entity := &TestEntity{Id: "123", Name: "My Name", ParentId: "ABC", Year: time.Now()}
				err := repo.saveOrUpdate("123", entity)
				Convey("Then saving the entity shouldn't return any errors", func() {
					So(err, ShouldBeNil)
				})

				Convey("And the number of entities should be 1", func() {
					count, _ := repo.CountAll()
					So(count, ShouldEqual, 1)
				})

				Convey("And the number of children should be 1", func() {
					children := make([]TestEntity, 0)
					err := repo.loadChildren("parent", "ABC", &children)
					So(err, ShouldBeNil)
					So(len(children), ShouldEqual, 1)
				})

				Convey("And this entity should be equal to the the saved one", func() {
					actualEntity, _ := repo.readEntity("123")
					So(actualEntity, shouldBeEqual, entity)
				})

			})

		})

		Convey("Given a table with one entity", func() {
			repo := createEmptyRepo()
			entity := &TestEntity{Id: "111", Name: "One Name", ParentId: "AAA"}
			repo.saveOrUpdate(entity.Id, entity)

			Convey("When I save an entity with a different Id", func() {
				newEntity := &TestEntity{Id: "222", Name: "Another Name", ParentId: "AAA"}
				repo.saveOrUpdate(newEntity.Id, newEntity)

				Convey("Then the number of entities should be 2", func() {
					count, _ := repo.CountAll()
					So(count, ShouldEqual, 2)
				})

			})

			Convey("When I save an entity with the same Id", func() {
				newEntity := &TestEntity{Id: "111", Name: "New Name", ParentId: "AAA"}
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
			repo := createEmptyRepo()
			for i := 1; i <= 3; i++ {
				e := &TestEntity{Id: strconv.Itoa(i), Name: fmt.Sprintf("Name %d", i), ParentId: "AAA"}
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
						So(e.ParentId, ShouldEqual, "AAA")
					}
				})
			})
			Convey("When I call GetAllIds", func() {
				ids, err := repo.getAllIds()
				Convey("Then It should not return any error", func() {
					So(err, ShouldBeNil)
				})
				Convey("And I get all saved ids", func() {
					So(len(ids), ShouldEqual, 3)
					for k := range ids {
						So(k, ShouldBeIn, []string{"1", "2", "3"})
					}
				})
			})

			Convey("When I call DeletaAll with one of the entities", func() {
				ids := make(map[string]bool)
				ids["1"] = true
				err := repo.DeleteAll(ids)
				Convey("Then It should not return any error", func() {
					So(err, ShouldBeNil)
				})
				Convey("Then CountAll should return 2", func() {
					count, _ := repo.CountAll()
					So(count, ShouldEqual, 2)
				})
				Convey("And the deleted record shouldn't be among the children", func() {
					children := make([]TestEntity, 0)
					err := repo.loadChildren("parent", "AAA", &children)
					So(err, ShouldBeNil)
					So(len(children), ShouldEqual, 2)
					for _, e := range children {
						So(e.Id, ShouldNotEqual, "1")
					}
				})

			})
		})
	})
}
