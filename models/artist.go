package models

import (
	"strings"
	"github.com/astaxie/beego"
)

type Artist struct {
	Id string
	Name string
}

func NoArticle(name string) string {
	articles := strings.Split(beego.AppConfig.String("ignoredArticles"), " ")
	for _, a := range articles {
		n := strings.TrimPrefix(name, a + " ")
		if (n != name) {
			return n
		}
	}
	return name
}