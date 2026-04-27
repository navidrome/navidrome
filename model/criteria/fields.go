package criteria

import "strings"

// FieldInfo contains semantic metadata about a criteria field
type FieldInfo struct {
	Name    string
	IsTag   bool
	IsRole  bool
	Numeric bool
	alias   string
}

var fieldMap = map[string]FieldInfo{
	"title":                {Name: "title"},
	"album":                {Name: "album"},
	"hascoverart":          {Name: "hascoverart"},
	"tracknumber":          {Name: "tracknumber"},
	"discnumber":           {Name: "discnumber"},
	"year":                 {Name: "year"},
	"date":                 {Name: "date", alias: "recordingdate"},
	"originalyear":         {Name: "originalyear"},
	"originaldate":         {Name: "originaldate"},
	"releaseyear":          {Name: "releaseyear"},
	"releasedate":          {Name: "releasedate"},
	"size":                 {Name: "size"},
	"compilation":          {Name: "compilation"},
	"missing":              {Name: "missing"},
	"explicitstatus":       {Name: "explicitstatus"},
	"dateadded":            {Name: "dateadded"},
	"datemodified":         {Name: "datemodified"},
	"discsubtitle":         {Name: "discsubtitle"},
	"comment":              {Name: "comment"},
	"lyrics":               {Name: "lyrics"},
	"sorttitle":            {Name: "sorttitle"},
	"sortalbum":            {Name: "sortalbum"},
	"sortartist":           {Name: "sortartist"},
	"sortalbumartist":      {Name: "sortalbumartist"},
	"albumcomment":         {Name: "albumcomment"},
	"catalognumber":        {Name: "catalognumber"},
	"filepath":             {Name: "filepath"},
	"filetype":             {Name: "filetype"},
	"codec":                {Name: "codec"},
	"duration":             {Name: "duration"},
	"bitrate":              {Name: "bitrate"},
	"bitdepth":             {Name: "bitdepth"},
	"samplerate":           {Name: "samplerate"},
	"bpm":                  {Name: "bpm"},
	"channels":             {Name: "channels"},
	"loved":                {Name: "loved"},
	"dateloved":            {Name: "dateloved"},
	"lastplayed":           {Name: "lastplayed"},
	"daterated":            {Name: "daterated"},
	"playcount":            {Name: "playcount"},
	"rating":               {Name: "rating"},
	"averagerating":        {Name: "averagerating", Numeric: true},
	"albumrating":          {Name: "albumrating"},
	"albumloved":           {Name: "albumloved"},
	"albumplaycount":       {Name: "albumplaycount"},
	"albumlastplayed":      {Name: "albumlastplayed"},
	"albumdateloved":       {Name: "albumdateloved"},
	"albumdaterated":       {Name: "albumdaterated"},
	"artistrating":         {Name: "artistrating"},
	"artistloved":          {Name: "artistloved"},
	"artistplaycount":      {Name: "artistplaycount"},
	"artistlastplayed":     {Name: "artistlastplayed"},
	"artistdateloved":      {Name: "artistdateloved"},
	"artistdaterated":      {Name: "artistdaterated"},
	"mbz_album_id":         {Name: "mbz_album_id"},
	"mbz_album_artist_id":  {Name: "mbz_album_artist_id"},
	"mbz_artist_id":        {Name: "mbz_artist_id"},
	"mbz_recording_id":     {Name: "mbz_recording_id"},
	"mbz_release_track_id": {Name: "mbz_release_track_id"},
	"mbz_release_group_id": {Name: "mbz_release_group_id"},
	"library_id":           {Name: "library_id", Numeric: true},

	// Backward compatibility: albumtype is an alias for the releasetype tag.
	"albumtype": {Name: "releasetype", IsTag: true},

	"random": {Name: "random"},
	"value":  {Name: "value"},
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
		fieldMap[name] = FieldInfo{Name: name, IsRole: true}
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
			fieldMap[name] = FieldInfo{Name: name, IsTag: true}
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
			fieldMap[name] = FieldInfo{Name: name, IsTag: true, Numeric: true}
		}
	}
}
