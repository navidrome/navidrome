package public

import (
	"net/http"
)

func (p *Router) handleDownloads(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(":id")
	if id == "" {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	err := p.archiver.ZipShare(r.Context(), id, w)
	checkShareError(r.Context(), w, err, id)
}
