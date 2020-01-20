package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
)

func FileServer(r chi.Router, fullPath, subPath string, root http.FileSystem) {
	if strings.ContainsAny(fullPath, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(fullPath, http.FileServer(justFilesFilesystem{root}))

	if subPath != "/" && subPath[len(subPath)-1] != '/' {
		r.Get(subPath, http.RedirectHandler(fullPath+"/", 302).ServeHTTP)
		subPath += "/"
	}
	subPath += "*"

	r.Get(subPath, func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}

type justFilesFilesystem struct {
	fs http.FileSystem
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return neuteredReaddirFile{f}, nil
}

type neuteredReaddirFile struct {
	http.File
}

func (f neuteredReaddirFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}
