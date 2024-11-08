package backgrounds

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/random"
	"gopkg.in/yaml.v3"
)

type Handler struct {
	list       []string
	lock       sync.RWMutex
	lastUpdate time.Time
}

func NewHandler() *Handler {
	h := &Handler{}
	go func() {
		_, _ = h.getImageList(log.NewContext(context.Background()))
	}()
	return h
}

const (
	imageHostingUrl = "https://unsplash.com/photos/%s/download?fm=jpg&w=1600&h=900&fit=max"
	imageListURL    = "https://www.navidrome.org/images/index.yml"
	imageListTTL    = 24 * time.Hour
)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	image, err := h.getRandomImage(r.Context())
	if err != nil {
		defaultImage, _ := base64.StdEncoding.DecodeString(consts.DefaultUILoginBackgroundOffline)
		w.Header().Set("content-type", "image/png")
		_, _ = w.Write(defaultImage)
		return
	}

	http.Redirect(w, r, imageURL(image), http.StatusFound)
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
	h.lock.RLock()
	if len(h.list) > 0 && time.Since(h.lastUpdate) < imageListTTL {
		defer h.lock.RUnlock()
		return h.list, nil
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
		return nil, err
	}
	defer resp.Body.Close()
	dec := yaml.NewDecoder(resp.Body)
	err = dec.Decode(&h.list)
	h.lastUpdate = time.Now()
	log.Debug(ctx, "Loaded background images from image service", "total", len(h.list), "elapsed", time.Since(start))
	return h.list, err
}

func imageURL(imageName string) string {
	imageName = strings.TrimSuffix(imageName, ".jpg")
	return fmt.Sprintf(imageHostingUrl, imageName)
}
