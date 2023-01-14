package utils

import (
	"strings"

	"github.com/navidrome/navidrome/conf"
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

func InsertString(slice []string, value string, index int) []string {
	return append(slice[:index], append([]string{value}, slice[index:]...)...)
}

func RemoveString(slice []string, index int) []string {
	return append(slice[:index], slice[index+1:]...)
}

func MoveString(slice []string, srcIndex int, dstIndex int) []string {
	value := slice[srcIndex]
	return InsertString(RemoveString(slice, srcIndex), value, dstIndex)
}

func BreakUpStringSlice(items []string, chunkSize int) [][]string {
	numTracks := len(items)
	var chunks [][]string
	for i := 0; i < numTracks; i += chunkSize {
		end := i + chunkSize
		if end > numTracks {
			end = numTracks
		}

		chunks = append(chunks, items[i:end])
	}
	return chunks
}

func RangeByChunks(items []string, chunkSize int, cb func([]string) error) error {
	chunks := BreakUpStringSlice(items, chunkSize)
	for _, chunk := range chunks {
		err := cb(chunk)
		if err != nil {
			return err
		}
	}
	return nil
}

func LongestCommonPrefix(list []string) string {
	if len(list) == 0 {
		return ""
	}

	for l := 0; l < len(list[0]); l++ {
		c := list[0][l]
		for i := 1; i < len(list); i++ {
			if l >= len(list[i]) || list[i][l] != c {
				return list[i][0:l]
			}
		}
	}
	return list[0]
}
