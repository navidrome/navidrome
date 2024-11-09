package backgrounds

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/random"
	"gopkg.in/yaml.v3"
)

type Handler struct {
	list       []string
	lock       sync.RWMutex
	lastUpdate time.Time
	cache      cache.FileCache
}

func NewHandler() *Handler {
	h := &Handler{}
	h.cache = cache.NewFileCache("backgrounds", "100MB", "backgrounds", 1000, h.serveImage)
	go func() {
		_ = h.loadImageList(log.NewContext(context.Background()))
	}()
	return h
}

const (
	imageHostingUrl = "https://unsplash.com/photos/%s/download?fm=jpg&w=1600&h=900&fit=max"
	imageListURL    = "https://www.navidrome.org/images/index.yml"
	imageListTTL    = 24 * time.Hour
)

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
	image := item.Key()
	if image == "" {
		return nil, errors.New("empty image name")
	}
	c := http.Client{
		Timeout: time.Minute,
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, imageURL(image), nil)
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (h *Handler) getRandomImage(ctx context.Context) (string, error) {
	err := h.loadImageList(ctx)
	if err != nil {
		return "", err
	}
	if len(h.list) == 0 {
		return "", errors.New("no images available")
	}
	rnd := random.Int64N(len(h.list))
	return h.list[rnd], nil
}

func (h *Handler) loadImageList(ctx context.Context) error {
	h.lock.RLock()
	if len(h.list) > 0 && time.Since(h.lastUpdate) < imageListTTL {
		h.lock.RUnlock()
		return nil
	}

	h.lock.RUnlock()
	h.lock.Lock()
	defer h.lock.Unlock()
	start := time.Now()

	c := http.Client{
		Timeout: time.Minute,
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, imageListURL, nil)
	resp, err := c.Do(req)
	if err != nil {
		log.Warn(ctx, "Could not get background images from image service", err)
		return err
	}
	defer resp.Body.Close()
	dec := yaml.NewDecoder(resp.Body)
	err = dec.Decode(&h.list)
	if err != nil {
		log.Warn(ctx, "Could not decode background images from image service", err)
		return err
	}
	h.lastUpdate = time.Now()
	log.Debug(ctx, "Loaded background images from image service", "total", len(h.list), "elapsed", time.Since(start))
	return nil
}

func imageURL(imageName string) string {
	imageName = strings.TrimSuffix(imageName, ".jpg")
	return fmt.Sprintf(imageHostingUrl, imageName)
}
