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
		MaxPreviewSize: 1024 * 1024,
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
	if imageResp["kind"] != "binary" {
		t.Fatalf("unexpected kind %q", imageResp["kind"])
	}
	if imageResp["mime"] != "image/png" {
		t.Fatalf("unexpected mime %q", imageResp["mime"])
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

	pdf := []byte("%PDF-1.4 test")
	if err := os.WriteFile(filepath.Join(root, "doc.pdf"), pdf, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err = http.Get(ts.URL + "/api/file?path=doc.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pdf file status %d", resp.StatusCode)
	}
	var pdfResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&pdfResp); err != nil {
		t.Fatal(err)
	}
	if pdfResp["kind"] != "binary" {
		t.Fatalf("unexpected pdf kind %q", pdfResp["kind"])
	}
	if pdfResp["mime"] != "application/pdf" {
		t.Fatalf("unexpected pdf mime %q", pdfResp["mime"])
	}

	resp, err = http.Get(ts.URL + "/api/raw?path=doc.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pdf raw status %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("unexpected pdf content type %q", resp.Header.Get("Content-Type"))
	}

	resp, err = http.Get(ts.URL + "/api/file?path=bin.dat")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("binary file status %d", resp.StatusCode)
	}
	var binResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&binResp); err != nil {
		t.Fatal(err)
	}
	if binResp["kind"] != "binary" {
		t.Fatalf("unexpected binary kind %q", binResp["kind"])
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
