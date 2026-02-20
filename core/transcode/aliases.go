package transcode

import (
	"slices"
	"strings"
)

// containerAliasGroups maps each container alias to a canonical group name.
var containerAliasGroups = func() map[string]string {
	groups := [][]string{
		{"aac", "adts", "m4a", "mp4", "m4b", "m4p"},
		{"mpeg", "mp3", "mp2"},
		{"ogg", "oga"},
		{"aif", "aiff"},
		{"asf", "wma"},
		{"mpc", "mpp"},
		{"wv"},
	}
	m := make(map[string]string)
	for _, g := range groups {
		canonical := g[0]
		for _, name := range g {
			m[name] = canonical
		}
	}
	return m
}()

// codecAliasGroups maps each codec alias to a canonical group name.
// Codecs within the same group are considered equivalent.
var codecAliasGroups = func() map[string]string {
	groups := [][]string{
		{"aac", "adts"},
		{"ac3", "ac-3"},
		{"eac3", "e-ac3", "e-ac-3", "eac-3"},
		{"mpc7", "musepack7"},
		{"mpc8", "musepack8"},
		{"wma1", "wmav1"},
		{"wma2", "wmav2"},
		{"wmalossless", "wma9lossless"},
		{"wmapro", "wma9pro"},
		{"shn", "shorten"},
		{"mp4als", "als"},
	}
	m := make(map[string]string)
	for _, g := range groups {
		for _, name := range g {
			m[name] = g[0] // canonical = first entry
		}
	}
	return m
}()

// matchesWithAliases checks if a value matches any entry in candidates,
// consulting the alias map for equivalent names.
func matchesWithAliases(value string, candidates []string, aliases map[string]string) bool {
	value = strings.ToLower(value)
	canonical := aliases[value]
	for _, c := range candidates {
		c = strings.ToLower(c)
		if c == value {
			return true
		}
		if canonical != "" && aliases[c] == canonical {
			return true
		}
	}
	return false
}

// matchesContainer checks if a file suffix matches any of the container names,
// including common aliases.
func matchesContainer(suffix string, containers []string) bool {
	return matchesWithAliases(suffix, containers, containerAliasGroups)
}

// matchesCodec checks if a codec matches any of the codec names,
// including common aliases.
func matchesCodec(codec string, codecs []string) bool {
	return matchesWithAliases(codec, codecs, codecAliasGroups)
}

func containsIgnoreCase(slice []string, s string) bool {
	return slices.ContainsFunc(slice, func(item string) bool {
		return strings.EqualFold(item, s)
	})
}
