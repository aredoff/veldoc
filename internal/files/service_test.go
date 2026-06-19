package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServiceReadAndTree(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "readme.md"), []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "note.txt"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	content, err := svc.ReadFile("readme.md")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Hello" {
		t.Fatalf("unexpected content: %q", content)
	}

	tree, err := svc.Tree()
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Children) < 2 {
		t.Fatalf("expected children, got %#v", tree.Children)
	}
}

func TestServicePathTraversal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc, err := NewService(root, 1024)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ReadFile("../outside.txt")
	if err != ErrOutsideRoot && err != ErrNotFound {
		t.Fatalf("expected outside root error, got %v", err)
	}
}

func TestServicePreviewSizeLimit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	data := make([]byte, 200)
	if err := os.WriteFile(filepath.Join(root, "big.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 100)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ReadForPreview("big.bin")
	if err != ErrTooLarge {
		t.Fatalf("expected too large for preview, got %v", err)
	}

	got, err := svc.ReadBytes("big.bin")
	if err != nil {
		t.Fatalf("expected download ok, got %v", err)
	}
	if len(got) != 200 {
		t.Fatalf("unexpected size %d", len(got))
	}
}

func TestServiceBinaryFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 1024)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ReadFile("bin.dat")
	if err != ErrBinaryFile {
		t.Fatalf("expected binary error, got %v", err)
	}
}

func TestServiceSymlinkOutsideRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 1024)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ReadFile("link.txt")
	if err != ErrOutsideRoot {
		t.Fatalf("expected outside root, got %v", err)
	}
}

func TestIsImage(t *testing.T) {
	t.Parallel()

	if !IsImage("photo.png") || !IsImage("photo.JPG") {
		t.Fatal("expected image")
	}
	if IsImage("note.txt") {
		t.Fatal("expected non-image")
	}
}

func TestServiceReadImageBytes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(root, "image.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	data, err := svc.ReadBytes("image.png")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != len(png) {
		t.Fatalf("unexpected size %d", len(data))
	}

	_, err = svc.ReadFile("image.png")
	if err != ErrBinaryFile {
		t.Fatalf("expected binary error for image text read, got %v", err)
	}
}

func TestServiceKnownBinaryTooLarge(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	data := make([]byte, 200)
	if err := os.WriteFile(filepath.Join(root, "big.png"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 100)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ReadFile("big.png")
	if err != ErrTooLarge {
		t.Fatalf("expected too large, got %v", err)
	}
}

func TestFileMIME(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"photo.png":  "image/png",
		"doc.pdf":    "application/pdf",
		"clip.mp4":   "video/mp4",
		"track.mp3":  "audio/mpeg",
		"unknown.zz": "application/octet-stream",
	}
	for path, want := range cases {
		if got := FileMIME(path); got != want {
			t.Fatalf("%s: expected %q, got %q", path, want, got)
		}
	}
}

func TestIsMarkdown(t *testing.T) {
	t.Parallel()

	if !IsMarkdown("doc.md") || !IsMarkdown("doc.markdown") {
		t.Fatal("expected markdown")
	}
	if IsMarkdown("doc.txt") {
		t.Fatal("expected non-markdown")
	}
}
