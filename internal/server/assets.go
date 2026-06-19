package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NewAssetVersion() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func injectAssetVersion(html, version string) string {
	return strings.ReplaceAll(html, "{{ASSET_VERSION}}", version)
}

func immutableCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
