package upload

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
	"unicode"
)

// Path constants for the default upload surface.
const (
	UploadPath  = "/goui/upload"
	FilesPrefix = "/goui/files"
)

var idPattern = regexp.MustCompile(`(?i)^[a-f0-9]{32}$`)

// Handler serves POST /goui/upload and GET /goui/files/:id over net/http.
type Handler struct {
	Store Storage
}

// NewHandler returns an http.Handler for upload and file download.
func NewHandler(store Storage) *Handler {
	return &Handler{Store: store}
}

// ServeHTTP routes upload and file requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Store == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not configured")
		return
	}

	switch {
	case r.Method == http.MethodPost && trimTrailingSlash(r.URL.Path) == UploadPath:
		h.handleUpload(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, FilesPrefix+"/"):
		h.handleFile(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Mount registers the handler on a ServeMux at the default paths.
func Mount(mux *http.ServeMux, store Storage) {
	h := NewHandler(store)
	mux.Handle(UploadPath, h)
	mux.Handle(FilesPrefix+"/", h)
}

func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "file required")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "file required")
		return
	}
	defer func() { _ = file.Close() }()

	ct := header.Header.Get("Content-Type")
	meta, err := h.Store.Save(header.Filename, ct, file, header.Size)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (h *Handler) handleFile(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	if !idPattern.MatchString(id) {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	rc, meta, err := h.Store.Open(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	defer func() { _ = rc.Close() }()

	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Disposition", contentDisposition(meta.Name))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

func contentDisposition(name string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r == '"' || r == '\\' || r == '\r' || r == '\n':
			return -1
		case unicode.IsControl(r):
			return -1
		default:
			return r
		}
	}, name)
	if safe == "" {
		safe = "file"
	}
	return `inline; filename="` + safe + `"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func trimTrailingSlash(p string) string {
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		return strings.TrimSuffix(p, "/")
	}
	return p
}
