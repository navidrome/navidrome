package capabilities

// SonicSimilarity provides audio-similarity based track discovery.
//
//nd:capability name=sonicsimilarity required=true
type SonicSimilarity interface {
	//nd:export name=nd_get_sonic_similar_tracks
	GetSonicSimilarTracks(GetSonicSimilarTracksRequest) (SonicSimilarityResponse, error)

	//nd:export name=nd_find_sonic_path
	FindSonicPath(FindSonicPathRequest) (SonicSimilarityResponse, error)
}

type GetSonicSimilarTracksRequest struct {
	Song  SongRef `json:"song"`
	Count int32   `json:"count"`
}

type FindSonicPathRequest struct {
	StartSong SongRef `json:"startSong"`
	EndSong   SongRef `json:"endSong"`
	Count     int32   `json:"count"`
}

type SonicSimilarityResponse struct {
	Matches []SonicMatch `json:"matches"`
}

type SonicMatch struct {
	Song       SongRef `json:"song"`
	Similarity float64 `json:"similarity"`
}
