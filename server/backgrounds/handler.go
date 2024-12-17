package backgrounds

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/random"
	"gopkg.in/yaml.v3"
)

const (
	//imageHostingUrl = "https://unsplash.com/photos/%s/download?fm=jpg&w=1600&h=900&fit=max"
	imageHostingUrl     = "https://www.navidrome.org/images/%s.jpg"
	imageListURL        = "https://www.navidrome.org/images/index.yml"
	imageListTTL        = 24 * time.Hour
	imageCacheDir       = "backgrounds"
	imageCacheSize      = "100MB"
	imageCacheMaxItems  = 1000
	imageRequestTimeout = 5 * time.Second
)

type Handler struct {
	httpClient *cache.HTTPClient
	cache      cache.FileCache
}

func NewHandler() *Handler {
	h := &Handler{}
	h.httpClient = cache.NewHTTPClient(&http.Client{Timeout: 5 * time.Second}, imageListTTL)
	h.cache = cache.NewFileCache(imageCacheDir, imageCacheSize, imageCacheDir, imageCacheMaxItems, h.serveImage)
	go func() {
		_, _ = h.getImageList(log.NewContext(context.Background()))
	}()
	return h
}

type cacheKey string

func (k cacheKey) Key() string {
	return string(k)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	image, err := h.getRandomImage(r.Context())
	if err != nil {
		h.serveDefaultImage(w)
		return
	}
	s, err := h.cache.Get(r.Context(), cacheKey(image))
	if err != nil {
		h.serveDefaultImage(w)
		return
	}
	defer s.Close()

	w.Header().Set("content-type", "image/jpeg")
	_, _ = io.Copy(w, s.Reader)
}

func (h *Handler) serveDefaultImage(w http.ResponseWriter) {
	defaultImage, _ := base64.StdEncoding.DecodeString(consts.DefaultUILoginBackgroundOffline)
	w.Header().Set("content-type", "image/png")
	_, _ = w.Write(defaultImage)
}

func (h *Handler) serveImage(ctx context.Context, item cache.Item) (io.Reader, error) {
	start := time.Now()
	image := item.Key()
	if image == "" {
		return nil, errors.New("empty image name")
	}
	c := http.Client{Timeout: imageRequestTimeout}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, imageURL(image), nil)
	resp, err := c.Do(req) //nolint:bodyclose // No need to close resp.Body, it will be closed via the CachedStream wrapper
	if errors.Is(err, context.DeadlineExceeded) {
		defaultImage, _ := base64.StdEncoding.DecodeString(consts.DefaultUILoginBackgroundOffline)
		return strings.NewReader(string(defaultImage)), nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get background image from hosting service: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code getting background image from hosting service: %d", resp.StatusCode)
	}
	log.Debug(ctx, "Got background image from hosting service", "image", image, "elapsed", time.Since(start))

	return resp.Body, nil
}

func (h *Handler) getRandomImage(ctx context.Context) (string, error) {
	list, err := h.getImageList(ctx)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", errors.New("no images available")
	}
	rnd := random.Int64N(len(list))
	return list[rnd], nil
}

func (h *Handler) getImageList(ctx context.Context) ([]string, error) {
	start := time.Now()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, imageListURL, nil)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Warn(ctx, "Could not get background images from image service", err)
		return nil, err
	}
	defer resp.Body.Close()

	var list []string
	dec := yaml.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	if err != nil {
		log.Warn(ctx, "Could not decode background images from image service", err)
		return nil, err
	}
	log.Debug(ctx, "Loaded background images from image service", "total", len(list), "elapsed", time.Since(start))
	return list, nil
}

func imageURL(imageName string) string {
	imageName = strings.TrimSuffix(imageName, ".jpg")
	return fmt.Sprintf(imageHostingUrl, imageName)
}
