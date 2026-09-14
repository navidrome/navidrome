package public

import (
	"io"
	"net/http"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/resources"
)

// handleAPlayerCSS serves the vendored APlayer CSS file
func (pub *Router) handleAPlayerCSS(w http.ResponseWriter, r *http.Request) {
	cssFile, err := resources.FS().Open("APlayer.min.css")
	if err != nil {
		log.Error(r.Context(), "Could not find APlayer.min.css", err)
		http.Error(w, "CSS file not found", http.StatusNotFound)
		return
	}
	defer cssFile.Close()

	cssContent, err := io.ReadAll(cssFile)
	if err != nil {
		log.Error(r.Context(), "Error reading APlayer.min.css", err)
		http.Error(w, "Error reading CSS file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	_, _ = w.Write(cssContent)
}

// handleAPlayerJS serves the vendored APlayer JavaScript file
func (pub *Router) handleAPlayerJS(w http.ResponseWriter, r *http.Request) {
	jsFile, err := resources.FS().Open("APlayer.min.js")
	if err != nil {
		log.Error(r.Context(), "Could not find APlayer.min.js", err)
		http.Error(w, "JS file not found", http.StatusNotFound)
		return
	}
	defer jsFile.Close()

	jsContent, err := io.ReadAll(jsFile)
	if err != nil {
		log.Error(r.Context(), "Error reading APlayer.min.js", err)
		http.Error(w, "Error reading JS file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	_, _ = w.Write(jsContent)
}
