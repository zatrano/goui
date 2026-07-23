package upload

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Meta describes a stored file.
type Meta struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	URL         string    `json:"url"`
	StoredAt    time.Time `json:"-"`
}

// Storage is a swappable file backend (local now; S3/MinIO later).
type Storage interface {
	Save(originalName, contentType string, r io.Reader, size int64) (Meta, error)
	Open(id string) (io.ReadCloser, Meta, error)
	Delete(id string) error
}

// LocalStore writes files under Dir and keeps an in-memory index.
type LocalStore struct {
	Dir      string
	BaseURL  string // e.g. /goui/files
	MaxBytes int64

	mu    sync.RWMutex
	index map[string]Meta
}

// NewLocalStore creates the directory if needed.
func NewLocalStore(dir, baseURL string, maxBytes int64) (*LocalStore, error) {
	if maxBytes <= 0 {
		maxBytes = 8 << 20 // 8 MiB
	}
	if baseURL == "" {
		baseURL = "/goui/files"
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &LocalStore{
		Dir:      dir,
		BaseURL:  strings.TrimRight(baseURL, "/"),
		MaxBytes: maxBytes,
		index:    map[string]Meta{},
	}, nil
}

func (s *LocalStore) Save(originalName, contentType string, r io.Reader, size int64) (Meta, error) {
	if size > 0 && size > s.MaxBytes {
		return Meta{}, errors.New("file too large")
	}
	id, err := newID()
	if err != nil {
		return Meta{}, err
	}
	safe := sanitizeName(originalName)
	ext := filepath.Ext(safe)
	diskName := id + ext
	path := filepath.Join(s.Dir, diskName)

	f, err := os.Create(path) //nolint:gosec // G304: path is Dir + generated id
	if err != nil {
		return Meta{}, err
	}
	defer func() { _ = f.Close() }()

	limited := io.LimitReader(r, s.MaxBytes+1)
	n, err := io.Copy(f, limited)
	if err != nil {
		_ = os.Remove(path)
		return Meta{}, err
	}
	if n > s.MaxBytes {
		_ = os.Remove(path)
		return Meta{}, errors.New("file too large")
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	meta := Meta{
		ID:          id,
		Name:        safe,
		ContentType: contentType,
		Size:        n,
		URL:         s.BaseURL + "/" + id,
		StoredAt:    time.Now().UTC(),
	}
	s.mu.Lock()
	s.index[id] = meta
	s.mu.Unlock()
	return meta, nil
}

func (s *LocalStore) Open(id string) (io.ReadCloser, Meta, error) {
	s.mu.RLock()
	meta, ok := s.index[id]
	s.mu.RUnlock()
	if !ok {
		return nil, Meta{}, os.ErrNotExist
	}
	matches, err := filepath.Glob(filepath.Join(s.Dir, id+".*"))
	if err != nil || len(matches) == 0 {
		// try exact id with no ext
		p := filepath.Join(s.Dir, id)
		f, err := os.Open(p) //nolint:gosec // G304: path is Dir + validated id
		if err != nil {
			return nil, Meta{}, err
		}
		return f, meta, nil
	}
	f, err := os.Open(matches[0]) //nolint:gosec // G304: path is Dir + validated id
	if err != nil {
		return nil, Meta{}, err
	}
	return f, meta, nil
}

func (s *LocalStore) Delete(id string) error {
	s.mu.Lock()
	delete(s.index, id)
	s.mu.Unlock()
	matches, _ := filepath.Glob(filepath.Join(s.Dir, id+".*"))
	for _, p := range matches {
		_ = os.Remove(p)
	}
	_ = os.Remove(filepath.Join(s.Dir, id))
	return nil
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func sanitizeName(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "..", "")
	if name == "" || name == "." {
		return "file"
	}
	return name
}

// MetaJSON is a helper for tests.
func MetaJSON(m Meta) string {
	b, _ := json.Marshal(m)
	return string(b)
}
