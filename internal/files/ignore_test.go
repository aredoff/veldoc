package files

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDocignore(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, ".docignore")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func treePaths(node Node) []string {
	var paths []string
	for _, child := range node.Children {
		paths = append(paths, child.Path)
		paths = append(paths, treePaths(child)...)
	}
	return paths
}

func hasPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}

func TestDocignoreRootPatterns(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDocignore(t, root, "*.log\nprivate/\n")
	if err := os.WriteFile(filepath.Join(root, "readme.md"), []byte("# Hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "debug.log"), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "private"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "private", "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := svc.Tree()
	if err != nil {
		t.Fatal(err)
	}
	paths := treePaths(tree)
	if hasPath(paths, "debug.log") {
		t.Fatal("debug.log should be ignored")
	}
	if hasPath(paths, "private") {
		t.Fatal("private/ should be ignored")
	}
	if !hasPath(paths, "readme.md") {
		t.Fatal("readme.md should be visible")
	}

	_, err = svc.ReadFile("debug.log")
	if err != ErrNotFound {
		t.Fatalf("expected not found for ignored file, got %v", err)
	}
}

func TestDocignoreNested(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "docs", "drafts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "published.md"), []byte("pub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "drafts", "wip.md"), []byte("wip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "other", "drafts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "other", "drafts", "note.md"), []byte("note"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeDocignore(t, filepath.Join(root, "docs"), "drafts/\n")

	svc, err := NewService(root, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := svc.Tree()
	if err != nil {
		t.Fatal(err)
	}
	paths := treePaths(tree)
	if hasPath(paths, "docs/drafts") {
		t.Fatal("docs/drafts should be ignored")
	}
	if !hasPath(paths, "other/drafts") {
		t.Fatal("other/drafts should be visible")
	}
	if !hasPath(paths, "docs/published.md") {
		t.Fatal("docs/published.md should be visible")
	}
}

func TestDocignoreNegation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDocignore(t, root, "*.log\n!important.log\n")
	if err := os.WriteFile(filepath.Join(root, "debug.log"), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "important.log"), []byte("important"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := svc.Tree()
	if err != nil {
		t.Fatal(err)
	}
	paths := treePaths(tree)
	if hasPath(paths, "debug.log") {
		t.Fatal("debug.log should be ignored")
	}
	if !hasPath(paths, "important.log") {
		t.Fatal("important.log should be visible via negation")
	}

	content, err := svc.ReadFile("important.log")
	if err != nil {
		t.Fatal(err)
	}
	if content != "important" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestDocignoreFileHidden(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeDocignore(t, root, "*.log\n")
	if err := os.WriteFile(filepath.Join(root, "readme.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(root, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := svc.Tree()
	if err != nil {
		t.Fatal(err)
	}
	if hasPath(treePaths(tree), ".docignore") {
		t.Fatal(".docignore should not appear in tree")
	}

	_, err = svc.ReadFile(".docignore")
	if err != ErrNotFound {
		t.Fatalf("expected not found for .docignore, got %v", err)
	}
}

func TestParentDirs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want []string
	}{
		{"readme.md", []string{""}},
		{"docs/file.md", []string{"", "docs"}},
		{"a/b/c.txt", []string{"", "a", "a/b"}},
	}
	for _, tc := range cases {
		got := parentDirs(tc.path)
		if len(got) != len(tc.want) {
			t.Fatalf("parentDirs(%q) = %v, want %v", tc.path, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("parentDirs(%q) = %v, want %v", tc.path, got, tc.want)
			}
		}
	}
}
