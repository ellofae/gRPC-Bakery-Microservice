package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/ellofae/RESTful-API-Gorilla/files"
	"github.com/gorilla/mux"
	hclog "github.com/hashicorp/go-hclog"
)

type FilesHandler struct {
	l     hclog.Logger
	store files.Storage
}

func NewFilesHandler(l hclog.Logger, s files.Storage) *FilesHandler {
	return &FilesHandler{l, s}
}

func (f *FilesHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	fn := vars["filename"]

	if id == "" || fn == "" {
		f.l.Error("Incorrect URI was given")
		http.Error(rw, "Incorrect URI", http.StatusBadRequest)
	}

	f.saveFile(id, fn, rw, r)
}

func (f *FilesHandler) saveFile(id string, path string, rw http.ResponseWriter, r *http.Request) {
	fp := filepath.Join(id, path)
	err := f.store.Save(fp, r.Body)
	if err != nil {
		f.l.Error("Didn't manage to save the file")
		http.Error(rw, "Not able to save a file", http.StatusInternalServerError)
	}
}
