package backgrounds

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

type Handler struct {
	list []string
	lock sync.RWMutex
}

func NewHandler() *Handler {
	h := &Handler{}
	go func() {
		_, _ = h.getImageList(context.Background())
	}()
	return h
}

const ndImageServiceURL = "https://www.navidrome.org"

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	image, err := h.getRandomImage(r.Context())
	if err != nil {
		defaultImage, _ := base64.StdEncoding.DecodeString(consts.DefaultUILoginBackgroundOffline)
		w.Header().Set("content-type", "image/png")
		_, _ = w.Write(defaultImage)
		return
	}

	http.Redirect(w, r, buildPath(ndImageServiceURL, "backgrounds", image+".jpg"), http.StatusFound)
}

func (h *Handler) getRandomImage(ctx context.Context) (string, error) {
	list, err := h.getImageList(ctx)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", errors.New("no images available")
	}
	rnd, _ := rand.Int(rand.Reader, big.NewInt(int64(len(list))))
	return list[rnd.Int64()], nil
}

func (h *Handler) getImageList(ctx context.Context) ([]string, error) {
	h.lock.RLock()
	if len(h.list) > 0 {
		defer h.lock.RUnlock()
		return h.list, nil
	}

	h.lock.RUnlock()
	h.lock.Lock()
	defer h.lock.Unlock()
	start := time.Now()

	c := http.Client{
		Timeout: 5 * time.Second,
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, buildPath(ndImageServiceURL, "images"), nil)
	resp, err := c.Do(req)
	if err != nil {
		log.Warn(ctx, "Could not get list from image service", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	h.list = strings.Split(string(body), "\n")
	log.Debug(ctx, "Loaded images from image service", "total", len(h.list), "elapsed", time.Since(start))
	return h.list, err
}

func buildPath(baseURL string, endpoint ...string) string {
	u, _ := url.Parse(baseURL)
	p := path.Join(endpoint...)
	u.Path = path.Join(u.Path, p)
	return u.String()
}
