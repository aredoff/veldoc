package files

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Sriram-PR/go-ignore"
)

const docignoreName = ".docignore"

func isDocignorePath(relPath string) bool {
	return filepath.Base(filepath.ToSlash(relPath)) == docignoreName
}

type ignoreMatcher struct {
	root    string
	matcher *ignore.Matcher
	loaded  map[string]struct{}
	mu      sync.Mutex
}

func newIgnoreMatcher(root string) *ignoreMatcher {
	return &ignoreMatcher{
		root:    root,
		matcher: ignore.New(),
		loaded:  make(map[string]struct{}),
	}
}

func (im *ignoreMatcher) ignored(relPath string, isDir bool) bool {
	im.loadForPath(relPath)
	rel := filepath.ToSlash(relPath)
	return im.matcher.Match(rel, isDir)
}

func (im *ignoreMatcher) loadForPath(relPath string) {
	dirs := parentDirs(relPath)
	im.mu.Lock()
	defer im.mu.Unlock()

	for _, dir := range dirs {
		if _, ok := im.loaded[dir]; ok {
			continue
		}
		im.loaded[dir] = struct{}{}

		path := filepath.Join(im.root, dir, docignoreName)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		im.matcher.AddPatterns(dir, data)
	}
}

func parentDirs(relPath string) []string {
	relPath = filepath.ToSlash(relPath)
	parts := strings.Split(relPath, "/")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}

	dirs := []string{""}
	current := ""
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		if current == "" {
			current = parts[i]
		} else {
			current = current + "/" + parts[i]
		}
		dirs = append(dirs, current)
	}
	return dirs
}
