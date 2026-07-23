package upload

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStore_SaveOpen(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(dir, "/goui/files", 1024)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := store.Save("hello.txt", "text/plain", bytes.NewReader([]byte("merhaba")), 7)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ID == "" || meta.URL != "/goui/files/"+meta.ID {
		t.Fatalf("%#v", meta)
	}
	rc, got, err := store.Open(meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "merhaba" || got.Name != "hello.txt" {
		t.Fatalf("data=%q meta=%#v", data, got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, meta.ID+".*"))
	if len(matches) != 1 {
		t.Fatalf("disk files: %v", matches)
	}
}

func TestLocalStore_TooLarge(t *testing.T) {
	store, _ := NewLocalStore(t.TempDir(), "", 4)
	_, err := store.Save("big.bin", "", bytes.NewReader([]byte("12345")), 5)
	if err == nil {
		t.Fatal("expected too large")
	}
}

func TestLocalStore_Delete(t *testing.T) {
	store, _ := NewLocalStore(t.TempDir(), "", 100)
	meta, _ := store.Save("a.txt", "text/plain", bytes.NewReader([]byte("x")), 1)
	_ = store.Delete(meta.ID)
	_, _, err := store.Open(meta.ID)
	if !os.IsNotExist(err) && err == nil {
		t.Fatal("expected missing")
	}
}
