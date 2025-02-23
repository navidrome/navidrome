package str

import (
	"strings"
	"unicode"

	"github.com/liuzl/gocc"
	"golang.org/x/text/unicode/norm"
)

func init() {
	var err error
	opencc, err = gocc.New("t2s")
	if err != nil {
		panic(err)
	}
}

var opencc *gocc.OpenCC

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

// NormalizeText performs normalization on the given text
// This includes
// - converts input to Unicode NFC
// - converts all Chinese character to simplified
func NormalizeText(s string) string {
	transformFuncs := []func(s string) string{
		norm.NFC.String,
		ToSimplifiedChinese,
	}

	for _, f := range transformFuncs {
		s = f(s)
	}

	return s
}

// ToSimplifiedChinese converts the given string from Traditional Chinese to Simplified
// Original string is returned if it contains no Chinese character
func ToSimplifiedChinese(s string) string {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			s, _ = opencc.Convert(s)
			break
		}
	}
	return s
}
