package models

import (
	"strings"
	"github.com/astaxie/beego"
)

type Artist struct {
	Id     string
	Name   string
	Albums map[string]bool
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

func (a *Artist) AddAlbums(albums ...interface{}) {
	if a.Albums == nil {
		a.Albums = make(map[string]bool)
	}
	for _, v := range albums {
		switch v := v.(type) {
		case *Album:
			a.Albums[v.Id] = true
		case map[string]bool:
			for k, _ := range v {
				a.Albums[k] = true
			}
		}
	}
}