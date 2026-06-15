package files

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("file not found")
	ErrOutsideRoot  = errors.New("path outside root")
	ErrBinaryFile   = errors.New("binary file")
	ErrTooLarge     = errors.New("file too large")
	ErrIsDirectory  = errors.New("path is a directory")
)

type NodeType string

const (
	NodeTypeDir  NodeType = "dir"
	NodeTypeFile NodeType = "file"
)

type Node struct {
	Path     string     `json:"path"`
	Name     string     `json:"name"`
	Type     NodeType   `json:"type"`
	Size     int64      `json:"size,omitempty"`
	Modified time.Time  `json:"modified"`
	Children []Node     `json:"children,omitempty"`
}

type Service struct {
	root        string
	maxFileSize int64
}

func NewService(root string, maxFileSize int64) (*Service, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("root directory: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("root path is not a directory")
	}

	return &Service{root: abs, maxFileSize: maxFileSize}, nil
}

func (s *Service) Root() string {
	return s.root
}

func (s *Service) Tree() (Node, error) {
	return s.buildTree(s.root, "")
}

func (s *Service) ReadFile(relPath string) (string, error) {
	if IsImage(relPath) {
		return "", ErrBinaryFile
	}

	data, err := s.ReadBytes(relPath)
	if err != nil {
		return "", err
	}
	if isBinary(data) {
		return "", ErrBinaryFile
	}

	return string(data), nil
}

func (s *Service) ReadBytes(relPath string) ([]byte, error) {
	abs, err := s.resolve(relPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, ErrIsDirectory
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrBinaryFile
	}
	if info.Size() > s.maxFileSize {
		return nil, ErrTooLarge
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, s.maxFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > s.maxFileSize {
		return nil, ErrTooLarge
	}

	return data, nil
}

func (s *Service) resolve(relPath string) (string, error) {
	clean := filepath.Clean("/" + strings.TrimPrefix(relPath, "/"))
	if clean == "/" || clean == "." {
		return "", ErrIsDirectory
	}

	rel := strings.TrimPrefix(clean, "/")
	abs := filepath.Join(s.root, rel)

	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(abs, s.root+string(os.PathSeparator)) && abs != s.root {
		return "", ErrOutsideRoot
	}

	info, err := os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(abs)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(abs), target)
		}
		target, err = filepath.Abs(target)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(target, s.root+string(os.PathSeparator)) && target != s.root {
			return "", ErrOutsideRoot
		}
		abs = target
	}

	return abs, nil
}

func (s *Service) buildTree(absPath, relPath string) (Node, error) {
	info, err := os.Lstat(absPath)
	if err != nil {
		return Node{}, err
	}

	node := Node{
		Path:     relPath,
		Name:     info.Name(),
		Type:     NodeTypeFile,
		Size:     info.Size(),
		Modified: info.ModTime(),
	}

	if !info.IsDir() {
		return node, nil
	}

	node.Type = NodeTypeDir
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return Node{}, err
	}

	for _, entry := range entries {
		if entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		childRel := entry.Name()
		if relPath != "" {
			childRel = filepath.ToSlash(filepath.Join(relPath, entry.Name()))
		}

		entryInfo, err := entry.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return Node{}, err
		}

		childAbs := filepath.Join(absPath, entry.Name())
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(childAbs)
			if err != nil {
				continue
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(childAbs), target)
			}
			target, err = filepath.Abs(target)
			if err != nil {
				continue
			}
			if !strings.HasPrefix(target, s.root+string(os.PathSeparator)) && target != s.root {
				continue
			}
			childAbs = target
		}

		child, err := s.buildTree(childAbs, childRel)
		if err != nil {
			return Node{}, err
		}
		node.Children = append(node.Children, child)
	}

	return node, nil
}

func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	for _, b := range sample {
		if b == 0 {
			return true
		}
	}
	return false
}

func IsMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

func IsImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".bmp":
		return true
	default:
		return false
	}
}

func ImageMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".bmp":
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}
