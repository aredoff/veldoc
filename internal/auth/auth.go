package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/aredoff/veldoc/internal/config"
)

const sessionCookie = "veldoc_session"

type Authenticator interface {
	Middleware(next http.Handler) http.Handler
	LoginHandler() http.Handler
	LogoutHandler() http.Handler
	LoginPage() http.Handler
}

type Options struct {
	AssetVersion string
}

func New(cfg config.Config, opts Options) (Authenticator, error) {
	switch cfg.Auth {
	case config.AuthNone:
		return &None{}, nil
	case config.AuthBasic:
		return &Basic{user: cfg.BasicUser, pass: cfg.BasicPass}, nil
	case config.AuthForm:
		return &Form{
			user:         cfg.FormUser,
			pass:         cfg.FormPass,
			secret:       []byte(cfg.SessionSecret),
			assetVersion: opts.AssetVersion,
		}, nil
	case config.AuthToken:
		return &Token{token: cfg.Token}, nil
	default:
		return nil, errors.New("unsupported auth mode")
	}
}

type None struct{}

func (n *None) Middleware(next http.Handler) http.Handler {
	return next
}

func (n *None) LoginHandler() http.Handler {
	return http.NotFoundHandler()
}

func (n *None) LogoutHandler() http.Handler {
	return http.NotFoundHandler()
}

func (n *None) LoginPage() http.Handler {
	return http.NotFoundHandler()
}

type Basic struct {
	user string
	pass string
}

func (b *Basic) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || !constantTimeEqual(user, b.user) || !constantTimeEqual(pass, b.pass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="veldoc"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (b *Basic) LoginHandler() http.Handler  { return http.NotFoundHandler() }
func (b *Basic) LogoutHandler() http.Handler { return http.NotFoundHandler() }
func (b *Basic) LoginPage() http.Handler     { return http.NotFoundHandler() }

type Token struct {
	token string
}

func (t *Token) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		value := strings.TrimPrefix(auth, "Bearer ")
		if !constantTimeEqual(value, t.token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (t *Token) LoginHandler() http.Handler  { return http.NotFoundHandler() }
func (t *Token) LogoutHandler() http.Handler { return http.NotFoundHandler() }
func (t *Token) LoginPage() http.Handler     { return http.NotFoundHandler() }

type Form struct {
	user         string
	pass         string
	secret       []byte
	assetVersion string
}

func (f *Form) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.validSession(r) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}

func (f *Form) LoginPage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.validSession(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		message := ""
		if r.URL.Query().Get("error") == "1" {
			message = "Invalid credentials"
		}
		html := strings.ReplaceAll(loginHTML, "{{ASSET_VERSION}}", f.assetVersion)
		html = strings.Replace(html, "{{ERROR}}", message, 1)
		_, _ = w.Write([]byte(html))
	})
}

func (f *Form) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		user := r.FormValue("username")
		pass := r.FormValue("password")
		if !constantTimeEqual(user, f.user) || !constantTimeEqual(pass, f.pass) {
			http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
			return
		}
		token, err := f.signSession()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int((24 * time.Hour).Seconds()),
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func (f *Form) LogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}

func (f *Form) validSession(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	return f.verifySession(cookie.Value)
}

func (f *Form) signSession() (string, error) {
	payload := base64.RawURLEncoding.EncodeToString([]byte("authenticated"))
	mac := hmac.New(sha256.New, f.secret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}

func (f *Form) verifySession(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	mac := hmac.New(sha256.New, f.secret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(parts[1])) == 1
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Veldoc Login</title>
  <link rel="icon" href="/favicon.ico" type="image/x-icon">
  <link rel="stylesheet" href="/static/login.css?v={{ASSET_VERSION}}">
</head>
<body>
  <form method="POST" action="/login">
    <h1>Veldoc</h1>
    <p class="error">{{ERROR}}</p>
    <label for="username">Username</label>
    <input id="username" name="username" autocomplete="username" required autofocus>
    <label for="password">Password</label>
    <input id="password" name="password" type="password" autocomplete="current-password" required>
    <button type="submit">Sign in</button>
  </form>
</body>
</html>`
