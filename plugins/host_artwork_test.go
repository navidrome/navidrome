//go:build !windows

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtworkService", Ordered, func() {
	var (
		manager *Manager
		tmpDir  string
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "artwork-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy the test-artwork plugin
		srcPath := filepath.Join(testdataDir, "test-artwork"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-artwork"+PackageExtension)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Compute SHA256 for the plugin
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = false
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

		// Initialize auth (required for token generation)
		ds := &tests.MockDataStore{MockedProperty: &tests.MockedPropertyRepo{}}
		auth.Init(ds)

		// Setup mock DataStore with pre-enabled plugin
		mockPluginRepo := tests.CreateMockPluginRepo()
		mockPluginRepo.Permitted = true
		mockPluginRepo.SetData(model.Plugins{{
			ID:      "test-artwork",
			Path:    destPath,
			SHA256:  hashHex,
			Enabled: true,
		}})
		dataStore := &tests.MockDataStore{
			MockedProperty: &tests.MockedPropertyRepo{},
			MockedPlugin:   mockPluginRepo,
		}

		// Create and start manager
		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             dataStore,
			subsonicRouter: http.NotFoundHandler(),
		}
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("Plugin Loading", func() {
		It("should load plugin with artwork permission", func() {
			manager.mu.RLock()
			p, ok := manager.plugins["test-artwork"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(p.manifest.Permissions).ToNot(BeNil())
			Expect(p.manifest.Permissions.Artwork).ToNot(BeNil())
		})
	})

	Describe("Artwork URL Generation", func() {
		type testArtworkInput struct {
			ArtworkType string `json:"artwork_type"`
			ID          string `json:"id"`
			Size        int32  `json:"size"`
		}
		type testArtworkOutput struct {
			URL   string  `json:"url,omitempty"`
			Error *string `json:"error,omitempty"`
		}

		callTestArtwork := func(ctx context.Context, artworkType, id string, size int32) (string, error) {
			manager.mu.RLock()
			p := manager.plugins["test-artwork"]
			manager.mu.RUnlock()

			instance, err := p.instance(ctx)
			if err != nil {
				return "", err
			}
			defer instance.Close(ctx)

			input := testArtworkInput{
				ArtworkType: artworkType,
				ID:          id,
				Size:        size,
			}
			inputBytes, _ := json.Marshal(input)
			_, outputBytes, err := instance.Call("nd_test_artwork", inputBytes)
			if err != nil {
				return "", err
			}

			var output testArtworkOutput
			if err := json.Unmarshal(outputBytes, &output); err != nil {
				return "", err
			}
			if output.Error != nil {
				return "", Errorf(*output.Error)
			}
			return output.URL, nil
		}

		It("should generate artist artwork URL", func() {
			url, err := callTestArtwork(GinkgoT().Context(), "artist", "ar-123", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(ContainSubstring("/img/"))
			Expect(url).ToNot(ContainSubstring("size="))

			// Decode JWT and verify artwork ID
			artID := decodeArtworkURL(url)
			Expect(artID.Kind).To(Equal(model.KindArtistArtwork))
			Expect(artID.ID).To(Equal("ar-123"))
		})

		It("should generate album artwork URL", func() {
			url, err := callTestArtwork(GinkgoT().Context(), "album", "al-456", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(ContainSubstring("/img/"))

			artID := decodeArtworkURL(url)
			Expect(artID.Kind).To(Equal(model.KindAlbumArtwork))
			Expect(artID.ID).To(Equal("al-456"))
		})

		It("should generate track artwork URL", func() {
			url, err := callTestArtwork(GinkgoT().Context(), "track", "mf-789", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(ContainSubstring("/img/"))

			artID := decodeArtworkURL(url)
			Expect(artID.Kind).To(Equal(model.KindMediaFileArtwork))
			Expect(artID.ID).To(Equal("mf-789"))
		})

		It("should generate playlist artwork URL", func() {
			url, err := callTestArtwork(GinkgoT().Context(), "playlist", "pl-abc", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(ContainSubstring("/img/"))

			artID := decodeArtworkURL(url)
			Expect(artID.Kind).To(Equal(model.KindPlaylistArtwork))
			Expect(artID.ID).To(Equal("pl-abc"))
		})

		It("should include size parameter when specified", func() {
			url, err := callTestArtwork(GinkgoT().Context(), "album", "al-456", 300)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(ContainSubstring("size=300"))

			artID := decodeArtworkURL(url)
			Expect(artID.Kind).To(Equal(model.KindAlbumArtwork))
			Expect(artID.ID).To(Equal("al-456"))
		})

		It("should handle unknown artwork type", func() {
			_, err := callTestArtwork(GinkgoT().Context(), "unknown", "id-123", 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown artwork type"))
		})
	})
})

// Errorf creates an error from a format string (helper for tests)
func Errorf(format string, args ...any) error {
	return &errorString{s: format}
}

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

// decodeArtworkURL extracts and decodes the JWT token from an artwork URL,
// returning the parsed ArtworkID. Panics on error (test helper).
func decodeArtworkURL(artworkURL string) model.ArtworkID {
	// URL format: http://localhost/img/<token>?size=...
	// Extract token from path after /img/
	idx := strings.Index(artworkURL, "/img/")
	Expect(idx).To(BeNumerically(">=", 0), "URL should contain /img/")

	tokenPart := artworkURL[idx+5:] // skip "/img/"
	// Remove query string if present
	if qIdx := strings.Index(tokenPart, "?"); qIdx >= 0 {
		tokenPart = tokenPart[:qIdx]
	}

	// Decode JWT token
	token, err := auth.TokenAuth.Decode(tokenPart)
	Expect(err).ToNot(HaveOccurred(), "Failed to decode JWT token")

	claims, err := token.AsMap(context.Background())
	Expect(err).ToNot(HaveOccurred(), "Failed to get claims from token")

	id, ok := claims["id"].(string)
	Expect(ok).To(BeTrue(), "Token should contain 'id' claim")

	artID, err := model.ParseArtworkID(id)
	Expect(err).ToNot(HaveOccurred(), "Failed to parse artwork ID from token")

	return artID
}
