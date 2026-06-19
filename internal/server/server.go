package server

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/aredoff/veldoc/internal/auth"
	"github.com/aredoff/veldoc/internal/config"
	"github.com/aredoff/veldoc/internal/files"
	"github.com/aredoff/veldoc/internal/markdown"
)

//go:embed web/*
var webFS embed.FS

type Server struct {
	cfg          config.Config
	files        *files.Service
	auth         auth.Authenticator
	logger       *slog.Logger
	assetVersion string
}

func New(cfg config.Config, fileService *files.Service, authenticator auth.Authenticator, logger *slog.Logger, assetVersion string) *Server {
	return &Server{
		cfg:          cfg,
		files:        fileService,
		auth:         authenticator,
		logger:       logger,
		assetVersion: assetVersion,
	}
}

func (s *Server) Handler() http.Handler {
	public := http.NewServeMux()
	public.HandleFunc("GET /healthz", s.handleHealth)
	public.Handle("GET /login", s.auth.LoginPage())
	public.Handle("POST /login", s.auth.LoginHandler())
	public.Handle("POST /logout", s.auth.LogoutHandler())

	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(webRoot))
	public.HandleFunc("GET /static/login.css", func(w http.ResponseWriter, _ *http.Request) {
		data, err := fs.ReadFile(webRoot, "login.css")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(data)
	})

	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/config", s.handleConfig)
	protected.HandleFunc("GET /api/tree", s.handleTree)
	protected.HandleFunc("GET /api/file", s.handleFile)
	protected.HandleFunc("GET /api/raw", s.handleRaw)
	protected.HandleFunc("GET /api/markdown", s.handleMarkdown)

	protected.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		data, err := fs.ReadFile(webRoot, "veldoc.ico")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/x-icon")
		_, _ = w.Write(data)
	})
	protected.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(webRoot, "index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		html := injectAssetVersion(string(data), s.assetVersion)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
	protected.Handle("GET /static/", immutableCache(http.StripPrefix("/static/", fileServer)))

	public.Handle("/", s.auth.Middleware(protected))
	return s.withMiddleware(public)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self' blob:; style-src 'self'; script-src 'self'; connect-src 'self' blob:; frame-src 'self' blob:; object-src 'self' blob:")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"pollIntervalMs": s.cfg.PollInterval.Milliseconds(),
		"auth":           s.cfg.Auth,
	})
}

func (s *Server) handleTree(w http.ResponseWriter, _ *http.Request) {
	tree, err := s.files.Tree()
	if err != nil {
		s.logger.Error("tree", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to read tree")
		return
	}
	writeJSON(w, tree)
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	content, err := s.files.ReadFile(path)
	if err == nil {
		writeJSON(w, map[string]string{
			"path":    path,
			"kind":    "text",
			"content": content,
		})
		return
	}
	if errors.Is(err, files.ErrBinaryFile) {
		writeJSON(w, map[string]string{
			"path": path,
			"kind": "binary",
			"mime": files.FileMIME(path),
			"url":  "/api/raw?path=" + url.QueryEscape(path),
		})
		return
	}

	status, msg := mapFileError(err)
	writeError(w, status, msg)
}

func (s *Server) handleRaw(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	var data []byte
	var err error
	if r.URL.Query().Get("download") == "1" {
		data, err = s.files.ReadBytes(path)
	} else {
		data, err = s.files.ReadForPreview(path)
	}
	if err != nil {
		status, msg := mapFileError(err)
		writeError(w, status, msg)
		return
	}

	w.Header().Set("Content-Type", files.FileMIME(path))
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(path))
	} else {
		w.Header().Set("Content-Disposition", "inline")
	}
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func (s *Server) handleMarkdown(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	if !files.IsMarkdown(path) {
		writeError(w, http.StatusBadRequest, "not a markdown file")
		return
	}

	content, err := s.files.ReadFile(path)
	if err != nil {
		status, msg := mapFileError(err)
		writeError(w, status, msg)
		return
	}

	html, err := markdown.Render(content)
	if err != nil {
		s.logger.Error("markdown", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to render markdown")
		return
	}

	writeJSON(w, map[string]string{
		"path": path,
		"html": html,
	})
}

func mapFileError(err error) (int, string) {
	switch {
	case errors.Is(err, files.ErrNotFound):
		return http.StatusNotFound, "file not found"
	case errors.Is(err, files.ErrOutsideRoot):
		return http.StatusForbidden, "access denied"
	case errors.Is(err, files.ErrBinaryFile):
		return http.StatusUnsupportedMediaType, "binary file"
	case errors.Is(err, files.ErrTooLarge):
		return http.StatusRequestEntityTooLarge, "file too large"
	case errors.Is(err, files.ErrIsDirectory):
		return http.StatusBadRequest, "path is a directory"
	default:
		return http.StatusInternalServerError, "failed to read file"
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

func (s *Server) ListenAndServe() error {
	srv := s.HTTPServer()
	s.logger.Info("listening", "addr", s.cfg.Addr, "root", s.files.Root(), "auth", s.cfg.Auth)
	return srv.ListenAndServe()
}
