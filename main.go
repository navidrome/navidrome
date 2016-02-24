package main

import (
	_ "github.com/deluan/gosonic/docs"
	_ "github.com/deluan/gosonic/routers"

	"github.com/astaxie/beego"
)

func main() {
	//// open a new index
	//itunes.LoadFolder("iTunes Music Library.xml")
	//
	//mapping := bleve.NewIndexMapping()
	//index, err := bleve.New("example.bleve", mapping)
	//if (err != nil) {
	//	index, err = bleve.Open("example.bleve")
	//}
	//
	//// index some data
	//doc := struct {
	//	Id    string
	//	Value string
	//}{
	//	Id: "01",
	//	Value: "deluan cotts quintao",
	//}
	//err = index.Index("01", doc)
	//fmt.Println(err)
	//
	//// search for some text
	//query := bleve.NewMatchQuery("*cotts*")
	//search := bleve.NewSearchRequest(query)
	//searchResults, err := index.Search(search)
	//fmt.Println(err)
	//fmt.Println(searchResults.Hits)

	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}
	beego.Run()
}
