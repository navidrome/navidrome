package str_test

import (
	"github.com/navidrome/navidrome/utils/str"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("NormalizeForFTS",
	func(expected string, values ...string) {
		Expect(str.NormalizeForFTS(values...)).To(Equal(expected))
	},
	Entry("strips dots and concatenates", "REM", "R.E.M."),
	Entry("strips slash", "ACDC", "AC/DC"),
	Entry("strips hyphen", "Aha", "A-ha"),
	Entry("skips unchanged ASCII words", "", "The Beatles"),
	Entry("handles mixed input", "REM", "R.E.M.", "Automatic for the People"),
	Entry("deduplicates", "REM", "R.E.M.", "R.E.M."),
	Entry("strips apostrophe from word", "N", "Guns N' Roses"),
	Entry("handles multiple values with punctuation", "REM ACDC", "R.E.M.", "AC/DC"),
	Entry("transliterates ø to o", "Bjork", "Bjørk"),
	Entry("transliterates Ø to O", "Oystein", "Øystein"),
	Entry("transliterates œ ligature to oe", "oeuvre", "œuvre"),
	Entry("transliterates Latin diacritics", "cafe", "café"),
	Entry("transliterates only the non-ASCII words", "Mo Ros", "Mø Rós"),
	Entry("combines punctuation strip and transliteration", "StEtienne St-Etienne", "St-Étienne"),
	Entry("deduplicates against punctuation form", "Cafe", "Café", "Cafe"),
	Entry("transliterates ß to ss", "Strasse", "Straße"),
)
