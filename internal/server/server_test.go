package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/aredoff/veldoc/internal/auth"
	"github.com/aredoff/veldoc/internal/config"
	"github.com/aredoff/veldoc/internal/files"
)

func TestHandlers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.md"), []byte("# Hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(root, "image.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}

	fileService, err := files.NewService(root, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	authenticator, err := auth.New(config.Config{Auth: config.AuthNone})
	if err != nil {
		t.Fatal(err)
	}

	srv := New(config.Config{
		Root:         root,
		Addr:         ":0",
		Auth:         config.AuthNone,
		PollInterval: 3000,
		MaxFileSize:  1024 * 1024,
	}, fileService, authenticator, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/api/tree")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tree status %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/api/file?path=hello.md")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var fileResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		t.Fatal(err)
	}
	if fileResp["content"] != "# Hi" {
		t.Fatalf("unexpected content %q", fileResp["content"])
	}

	resp, err = http.Get(ts.URL + "/api/file?path=image.png")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var imageResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&imageResp); err != nil {
		t.Fatal(err)
	}
	if imageResp["kind"] != "image" {
		t.Fatalf("unexpected kind %q", imageResp["kind"])
	}

	resp, err = http.Get(ts.URL + "/api/raw?path=image.png")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("raw status %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	resp, err = http.Get(ts.URL + "/api/markdown?path=hello.md")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("markdown status %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("index status %d", resp.StatusCode)
	}
}
