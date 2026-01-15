// Test plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-metadata-agent.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"errors"
	"strconv"

	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

func init() {
	metadata.Register(&testMetadataAgent{})
}

type testMetadataAgent struct{}

// checkConfigError checks if the plugin is configured to return an error.
// If "error" config is set, it returns an error with that message.
func checkConfigError() error {
	errMsg, hasErr := pdk.GetConfig("error")
	if !hasErr || errMsg == "" {
		return nil
	}
	return errors.New(errMsg)
}

func (t *testMetadataAgent) GetArtistMBID(input metadata.ArtistMBIDRequest) (*metadata.ArtistMBIDResponse, error) {
	if err := checkConfigError(); err != nil {
		return nil, err
	}
	return &metadata.ArtistMBIDResponse{MBID: "test-mbid-" + input.Name}, nil
}

func (t *testMetadataAgent) GetArtistURL(input metadata.ArtistRequest) (*metadata.ArtistURLResponse, error) {
	if err := checkConfigError(); err != nil {
		return nil, err
	}
	return &metadata.ArtistURLResponse{URL: "https://test.example.com/artist/" + input.Name}, nil
}

func (t *testMetadataAgent) GetArtistBiography(input metadata.ArtistRequest) (*metadata.ArtistBiographyResponse, error) {
	if err := checkConfigError(); err != nil {
		return nil, err
	}
	return &metadata.ArtistBiographyResponse{Biography: "Biography for " + input.Name}, nil
}

func (t *testMetadataAgent) GetArtistImages(input metadata.ArtistRequest) (*metadata.ArtistImagesResponse, error) {
	if err := checkConfigError(); err != nil {
		return nil, err
	}
	return &metadata.ArtistImagesResponse{
		Images: []metadata.ImageInfo{
			{URL: "https://test.example.com/images/" + input.Name + "/large.jpg", Size: 500},
			{URL: "https://test.example.com/images/" + input.Name + "/small.jpg", Size: 100},
		},
	}, nil
}

func (t *testMetadataAgent) GetSimilarArtists(input metadata.SimilarArtistsRequest) (*metadata.SimilarArtistsResponse, error) {
	if err := checkConfigError(); err != nil {
		return nil, err
	}
	limit := int(input.Limit)
	if limit == 0 {
		limit = 5
	}
	artists := make([]metadata.ArtistRef, 0, limit)
	for i := range limit {
		artists = append(artists, metadata.ArtistRef{
			ID:   "similar-artist-id-" + strconv.Itoa(i+1),
			Name: input.Name + " Similar " + string(rune('A'+i)),
			MBID: "similar-mbid-" + strconv.Itoa(i+1),
		})
	}
	return &metadata.SimilarArtistsResponse{Artists: artists}, nil
}

func (t *testMetadataAgent) GetArtistTopSongs(input metadata.TopSongsRequest) (*metadata.TopSongsResponse, error) {
	if err := checkConfigError(); err != nil {
		return nil, err
	}
	count := int(input.Count)
	if count == 0 {
		count = 5
	}
	songs := make([]metadata.SongRef, 0, count)
	for i := range count {
		songs = append(songs, metadata.SongRef{
			ID:   "song-id-" + strconv.Itoa(i+1),
			Name: input.Name + " Song " + strconv.Itoa(i+1),
			MBID: "song-mbid-" + strconv.Itoa(i+1),
		})
	}
	return &metadata.TopSongsResponse{Songs: songs}, nil
}

func (t *testMetadataAgent) GetAlbumInfo(input metadata.AlbumRequest) (*metadata.AlbumInfoResponse, error) {
	if err := checkConfigError(); err != nil {
		return nil, err
	}
	return &metadata.AlbumInfoResponse{
		Name:        input.Name,
		MBID:        "test-album-mbid-" + input.Name,
		Description: "Description for " + input.Name + " by " + input.Artist,
		URL:         "https://test.example.com/album/" + input.Name,
	}, nil
}

func (t *testMetadataAgent) GetAlbumImages(input metadata.AlbumRequest) (*metadata.AlbumImagesResponse, error) {
	if err := checkConfigError(); err != nil {
		return nil, err
	}
	return &metadata.AlbumImagesResponse{
		Images: []metadata.ImageInfo{
			{URL: "https://test.example.com/albums/" + input.Name + "/cover.jpg", Size: 500},
		},
	}, nil
}

func main() {}
