package criteria

import "strings"

// FieldInfo contains semantic metadata about a criteria field.
type FieldInfo struct {
	Alias   string // If set, this field is a backward-compat alias for another canonical name
	IsTag   bool
	IsRole  bool
	Numeric bool
	Boolean bool

	tagAlias string // If set, a tag name from mappings.yml that resolves to this field
	name     string // Canonical name, populated by LookupField from the map key
}

// Name returns the canonical field name (the map key used to register this field).
func (f FieldInfo) Name() string {
	return f.name
}

var fieldMap = map[string]FieldInfo{
	"title":                {},
	"album":                {},
	"hascoverart":          {Boolean: true},
	"tracknumber":          {},
	"discnumber":           {},
	"year":                 {},
	"date":                 {tagAlias: "recordingdate"},
	"originalyear":         {},
	"originaldate":         {},
	"releaseyear":          {},
	"releasedate":          {},
	"size":                 {},
	"compilation":          {Boolean: true},
	"missing":              {Boolean: true},
	"explicitstatus":       {},
	"dateadded":            {},
	"datemodified":         {},
	"discsubtitle":         {},
	"comment":              {},
	"lyrics":               {},
	"sorttitle":            {},
	"sortalbum":            {},
	"sortartist":           {},
	"sortalbumartist":      {},
	"albumcomment":         {},
	"catalognumber":        {},
	"filepath":             {},
	"filetype":             {},
	"codec":                {},
	"duration":             {},
	"bitrate":              {},
	"bitdepth":             {},
	"samplerate":           {},
	"bpm":                  {},
	"channels":             {},
	"loved":                {Boolean: true},
	"dateloved":            {},
	"lastplayed":           {},
	"daterated":            {},
	"playcount":            {},
	"rating":               {},
	"averagerating":        {Numeric: true},
	"albumrating":          {},
	"albumloved":           {Boolean: true},
	"albumplaycount":       {},
	"albumlastplayed":      {},
	"albumdateloved":       {},
	"albumdaterated":       {},
	"artistrating":         {},
	"artistloved":          {Boolean: true},
	"artistplaycount":      {},
	"artistlastplayed":     {},
	"artistdateloved":      {},
	"artistdaterated":      {},
	"mbz_album_id":         {},
	"mbz_album_artist_id":  {},
	"mbz_artist_id":        {},
	"mbz_recording_id":     {},
	"mbz_release_track_id": {},
	"mbz_release_group_id": {},
	"rgalbumgain":          {Numeric: true},
	"rgalbumpeak":          {Numeric: true},
	"rgtrackgain":          {Numeric: true},
	"rgtrackpeak":          {Numeric: true},
	"library_id":           {Numeric: true},

	// Backward compatibility: albumtype is an alias for the releasetype tag.
	"albumtype": {Alias: "releasetype", IsTag: true},

	// Pseudo-field for random sorting
	"random": {},
}

// AllFieldNames returns the names of all registered criteria fields.
func AllFieldNames() []string {
	names := make([]string, 0, len(fieldMap))
	for name := range fieldMap {
		names = append(names, name)
	}
	return names
}

// LookupField returns semantic metadata for a criteria field name.
func LookupField(name string) (FieldInfo, bool) {
	key := strings.ToLower(name)
	f, ok := fieldMap[key]
	if ok {
		if f.Alias != "" {
			f.name = f.Alias
		} else {
			f.name = key
		}
	}
	return f, ok
}

// AddRoles adds roles to the field map. This is used to add all artist roles to the field map, so they can be used in
// smart playlists.
func AddRoles(roles []string) {
	for _, role := range roles {
		name := strings.ToLower(role)
		if _, ok := fieldMap[name]; ok {
			continue
		}
		fieldMap[name] = FieldInfo{IsRole: true}
	}
}

// AddTagNames adds tag names to the field map. This is used to add all tags mapped in the `mappings.yml`
// configuration file.
func AddTagNames(tagNames []string) {
	for _, tagName := range tagNames {
		name := strings.ToLower(tagName)
		if _, ok := fieldMap[name]; ok {
			continue
		}
		for key, fm := range fieldMap {
			if fm.tagAlias == name {
				fm.Alias = key
				fm.tagAlias = ""
				fieldMap[name] = fm
				break
			}
		}
		if _, ok := fieldMap[name]; !ok {
			fieldMap[name] = FieldInfo{IsTag: true}
		}
	}
}

// AddNumericTags adds tags that should be treated as numbers.
func AddNumericTags(tagNames []string) {
	for _, tagName := range tagNames {
		name := strings.ToLower(tagName)
		if fm, ok := fieldMap[name]; ok {
			fm.Numeric = true
			fieldMap[name] = fm
		} else {
			fieldMap[name] = FieldInfo{IsTag: true, Numeric: true}
		}
	}
}
