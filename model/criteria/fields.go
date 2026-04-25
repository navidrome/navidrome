package criteria

import "strings"

// FieldInfo describes a criteria field without tying it to persistence details.
type FieldInfo struct {
	Name    string
	IsTag   bool
	IsRole  bool
	Numeric bool
}

var fieldMap = map[string]*fieldMetadata{
	"title":                {name: "title"},
	"album":                {name: "album"},
	"hascoverart":          {name: "hascoverart"},
	"tracknumber":          {name: "tracknumber"},
	"discnumber":           {name: "discnumber"},
	"year":                 {name: "year"},
	"date":                 {name: "date", alias: "recordingdate"},
	"originalyear":         {name: "originalyear"},
	"originaldate":         {name: "originaldate"},
	"releaseyear":          {name: "releaseyear"},
	"releasedate":          {name: "releasedate"},
	"size":                 {name: "size"},
	"compilation":          {name: "compilation"},
	"missing":              {name: "missing"},
	"explicitstatus":       {name: "explicitstatus"},
	"dateadded":            {name: "dateadded"},
	"datemodified":         {name: "datemodified"},
	"discsubtitle":         {name: "discsubtitle"},
	"comment":              {name: "comment"},
	"lyrics":               {name: "lyrics"},
	"sorttitle":            {name: "sorttitle"},
	"sortalbum":            {name: "sortalbum"},
	"sortartist":           {name: "sortartist"},
	"sortalbumartist":      {name: "sortalbumartist"},
	"albumcomment":         {name: "albumcomment"},
	"catalognumber":        {name: "catalognumber"},
	"filepath":             {name: "filepath"},
	"filetype":             {name: "filetype"},
	"codec":                {name: "codec"},
	"duration":             {name: "duration"},
	"bitrate":              {name: "bitrate"},
	"bitdepth":             {name: "bitdepth"},
	"samplerate":           {name: "samplerate"},
	"bpm":                  {name: "bpm"},
	"channels":             {name: "channels"},
	"loved":                {name: "loved"},
	"dateloved":            {name: "dateloved"},
	"lastplayed":           {name: "lastplayed"},
	"daterated":            {name: "daterated"},
	"playcount":            {name: "playcount"},
	"rating":               {name: "rating"},
	"averagerating":        {name: "averagerating", numeric: true},
	"albumrating":          {name: "albumrating"},
	"albumloved":           {name: "albumloved"},
	"albumplaycount":       {name: "albumplaycount"},
	"albumlastplayed":      {name: "albumlastplayed"},
	"albumdateloved":       {name: "albumdateloved"},
	"albumdaterated":       {name: "albumdaterated"},
	"artistrating":         {name: "artistrating"},
	"artistloved":          {name: "artistloved"},
	"artistplaycount":      {name: "artistplaycount"},
	"artistlastplayed":     {name: "artistlastplayed"},
	"artistdateloved":      {name: "artistdateloved"},
	"artistdaterated":      {name: "artistdaterated"},
	"mbz_album_id":         {name: "mbz_album_id"},
	"mbz_album_artist_id":  {name: "mbz_album_artist_id"},
	"mbz_artist_id":        {name: "mbz_artist_id"},
	"mbz_recording_id":     {name: "mbz_recording_id"},
	"mbz_release_track_id": {name: "mbz_release_track_id"},
	"mbz_release_group_id": {name: "mbz_release_group_id"},
	"library_id":           {name: "library_id", numeric: true},

	// Backward compatibility: albumtype is an alias for the releasetype tag.
	"albumtype": {name: "releasetype", isTag: true},

	"random": {name: "random"},
	"value":  {name: "value"},
}

type fieldMetadata struct {
	name    string
	isRole  bool
	isTag   bool
	alias   string
	numeric bool
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
	f, ok := fieldMap[strings.ToLower(name)]
	if !ok {
		return FieldInfo{}, false
	}
	return FieldInfo{
		Name:    f.name,
		IsTag:   f.isTag,
		IsRole:  f.isRole,
		Numeric: f.numeric,
	}, true
}

// AddRoles adds roles to the field map. This is used to add all artist roles to the field map, so they can be used in
// smart playlists.
func AddRoles(roles []string) {
	for _, role := range roles {
		name := strings.ToLower(role)
		if _, ok := fieldMap[name]; ok {
			continue
		}
		fieldMap[name] = &fieldMetadata{name: name, isRole: true}
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
		for _, fm := range fieldMap {
			if fm.alias == name {
				fieldMap[name] = fm
				break
			}
		}
		if _, ok := fieldMap[name]; !ok {
			fieldMap[name] = &fieldMetadata{name: name, isTag: true}
		}
	}
}

// AddNumericTags adds tags that should be treated as numbers.
func AddNumericTags(tagNames []string) {
	for _, tagName := range tagNames {
		name := strings.ToLower(tagName)
		if fm, ok := fieldMap[name]; ok {
			fm.numeric = true
		} else {
			fieldMap[name] = &fieldMetadata{name: name, isTag: true, numeric: true}
		}
	}
}
