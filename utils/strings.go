package utils

import (
	"strings"

	"github.com/deluan/navidrome/conf"
)

func NoArticle(name string) string {
	articles := strings.Split(conf.Server.IgnoredArticles, " ")
	for _, a := range articles {
		n := strings.TrimPrefix(name, a+" ")
		if n != name {
			return n
		}
	}
	return name
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func InsertString(array []string, value string, index int) []string {
	return append(array[:index], append([]string{value}, array[index:]...)...)
}

func RemoveString(array []string, index int) []string {
	return append(array[:index], array[index+1:]...)
}

func MoveString(array []string, srcIndex int, dstIndex int) []string {
	value := array[srcIndex]
	return InsertString(RemoveString(array, srcIndex), value, dstIndex)
}
