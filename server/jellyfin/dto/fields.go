package dto

import "strings"

// Fields is the parsed set of a Jellyfin request's Fields param (lowercased). It controls which
// conditional fields a mapped item carries — chiefly MediaSources — matching real Jellyfin, which
// omits those unless the client asks for them.
type Fields map[string]struct{}

// ParseFields splits the comma-separated Fields param into a lowercased set.
func ParseFields(csv string) Fields {
	f := Fields{}
	for name := range strings.SplitSeq(csv, ",") {
		if name = strings.TrimSpace(strings.ToLower(name)); name != "" {
			f[name] = struct{}{}
		}
	}
	return f
}

func (f Fields) Has(name string) bool {
	_, ok := f[strings.ToLower(name)]
	return ok
}
