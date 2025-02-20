package str

import (
	"strings"
)

var utf8ToAscii = func() *strings.Replacer {
	var utf8Map = map[string]string{
		"'": `‘’‛′`,
		`"`: `＂〃ˮײ᳓″‶˶ʺ“”˝‟`,
		"-": `‐–—−―`,
	}

	list := make([]string, 0, len(utf8Map)*2)
	for ascii, utf8 := range utf8Map {
		for _, r := range utf8 {
			list = append(list, string(r), ascii)
		}
	}
	return strings.NewReplacer(list...)
}()

func Clear(name string) string {
	return utf8ToAscii.Replace(name)
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
