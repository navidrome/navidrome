package repositories

import (
	. "github.com/smartystreets/goconvey/convey"
	_ "github.com/deluan/gosonic/tests"
	"testing"
)

const (
	testCollectionName = "TestCollection"
)

func TestCreateCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	dbInstance().Drop(testCollectionName)

	Convey("Given an empty DB", t, func() {

		Convey("When creating a new collection", func() {
			newCol := createCollection(testCollectionName)

			Convey("Then it should create the collection", func() {
				So(dbInstance().Use(testCollectionName), ShouldNotBeNil)
			})
			Convey("And it should create a default index on Id", func() {
				allIndexes := newCol.AllIndexes()
				So(len(allIndexes), ShouldEqual, 1)
				So(len(allIndexes[0]), ShouldEqual, 1)
				So(allIndexes[0], ShouldContain, "Id")
			})
		})

		Convey("When creating a new collection with a 'Name' index", func() {
			newCol := createCollection(testCollectionName, "Name")

			Convey("Then it should create a default index [Id] and an index on 'Name'", func() {
				listOfIndexes := newCol.AllIndexes()

				So(len(listOfIndexes), ShouldEqual, 2)

				var allPaths = make(map[string]bool)
				for _, i := range listOfIndexes {
					allPaths[i[0]] = true
				}

				So(allPaths, ShouldContainKey, "Id")
				So(allPaths, ShouldContainKey, "Name")
			})
		})
	})
}