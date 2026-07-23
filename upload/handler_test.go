package upload

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandler_UploadAndDownload(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(dir, FilesPrefix, 1<<20)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}

	mux := http.NewServeMux()
	Mount(mux, store)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", `report".txt`)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := io.WriteString(part, "hello goui"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, UploadPath, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("upload status = %d, body=%s", rr.Code, rr.Body.String())
	}

	var meta Meta
	if err := json.Unmarshal(rr.Body.Bytes(), &meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if meta.ID == "" || meta.URL == "" {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	getReq := httptest.NewRequest(http.MethodGet, meta.URL, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("download status = %d", getRR.Code)
	}
	if getRR.Body.String() != "hello goui" {
		t.Fatalf("body = %q", getRR.Body.String())
	}
	cd := getRR.Header().Get("Content-Disposition")
	if strings.Contains(cd, `"`) && strings.Contains(cd, `report".txt`) {
		t.Fatalf("unsafe Content-Disposition: %q", cd)
	}
	if !strings.Contains(cd, "reporttxt") && !strings.Contains(cd, "report") {
		t.Fatalf("Content-Disposition missing sanitized name: %q", cd)
	}
}

func TestHandler_MissingFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(dir, FilesPrefix, 1<<20)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	h := NewHandler(store)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, UploadPath, nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandler_InvalidID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(dir, FilesPrefix, 1<<20)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	h := NewHandler(store)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, FilesPrefix+"/../etc/passwd", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestContentDisposition_StripsInjection(t *testing.T) {
	got := contentDisposition("a\"\r\nb.txt")
	if strings.Contains(got, "\r") || strings.Contains(got, "\n") || strings.Contains(got, `a"`) {
		t.Fatalf("unsafe disposition: %q", got)
	}
}

func TestLocalStore_Persists(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(dir, FilesPrefix, 1<<20)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	meta, err := store.Save("note.txt", "text/plain", strings.NewReader("x"), 1)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, meta.ID+".*"))
	if len(matches) == 0 {
		if _, err := os.Stat(filepath.Join(dir, meta.ID)); err != nil {
			t.Fatalf("expected stored file: %v", err)
		}
	}
}
