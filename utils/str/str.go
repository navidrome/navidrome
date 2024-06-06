package str

import (
	"strings"
)

var utf8ToAscii = strings.NewReplacer(
	"–", "-",
	"‐", "-",
	"“", `"`,
	"”", `"`,
	"‘", `'`,
	"’", `'`,
)

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
