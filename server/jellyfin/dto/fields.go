package dto

import "strings"

// Fields is the parsed set of a Jellyfin request's Fields param (lowercased). It controls which
// conditional fields a mapped item carries — chiefly MediaSources — matching real Jellyfin, which
// omits those unless the client asks for them.
type Fields map[string]struct{}

// ParseFields builds a lowercased set from the Fields param. It accepts each value comma-separated
// (Fields=a,b) and across repeated params (Fields=a&Fields=b), both of which real Jellyfin honors.
func ParseFields(values ...string) Fields {
	f := Fields{}
	for _, csv := range values {
		for name := range strings.SplitSeq(csv, ",") {
			if name = strings.TrimSpace(strings.ToLower(name)); name != "" {
				f[name] = struct{}{}
			}
		}
	}
	return f
}

func (f Fields) Has(name string) bool {
	_, ok := f[strings.ToLower(name)]
	return ok
}
