package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type AuthMode string

const (
	AuthNone  AuthMode = "none"
	AuthBasic AuthMode = "basic"
	AuthForm  AuthMode = "form"
	AuthToken AuthMode = "token"
)

type Config struct {
	Root         string
	Addr         string
	Auth         AuthMode
	BasicUser    string
	BasicPass    string
	FormUser     string
	FormPass     string
	SessionSecret string
	Token        string
	PollInterval time.Duration
	MaxFileSize  int64
}

func Load() (Config, error) {
	var cfg Config

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	root := fs.String("root", envOr("VELDOC_ROOT", "/data"), "root directory to serve")
	addr := fs.String("addr", envOr("VELDOC_ADDR", ":8080"), "listen address")
	auth := fs.String("auth", envOr("VELDOC_AUTH", "none"), "auth mode: none, basic, form, token")
	formUser := fs.String("form-user", envOr("VELDOC_FORM_USER", ""), "form auth username")
	formPass := fs.String("form-password", envOr("VELDOC_FORM_PASSWORD", ""), "form auth password")
	sessionSecret := fs.String("session-secret", envOr("VELDOC_SESSION_SECRET", ""), "form auth session secret")
	pollInterval := fs.String("poll-interval", envOr("VELDOC_POLL_INTERVAL", "3s"), "client poll interval hint")
	maxFileSize := fs.String("max-file-size", envOr("VELDOC_MAX_FILE_SIZE", "2097152"), "max file size in bytes")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}

	cfg.Root = *root
	cfg.Addr = *addr
	cfg.Auth = AuthMode(strings.ToLower(strings.TrimSpace(*auth)))
	cfg.BasicUser = envOr("VELDOC_BASIC_USER", "")
	cfg.BasicPass = envOr("VELDOC_BASIC_PASSWORD", "")
	cfg.FormUser = *formUser
	cfg.FormPass = *formPass
	cfg.SessionSecret = *sessionSecret
	cfg.Token = envOr("VELDOC_TOKEN", "")

	interval, err := time.ParseDuration(*pollInterval)
	if err != nil {
		return Config{}, fmt.Errorf("invalid poll interval: %w", err)
	}
	cfg.PollInterval = interval

	size, err := parseSize(*maxFileSize)
	if err != nil {
		return Config{}, fmt.Errorf("invalid max file size: %w", err)
	}
	cfg.MaxFileSize = size

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.Root == "" {
		return errors.New("root directory is required")
	}
	if c.Addr == "" {
		return errors.New("listen address is required")
	}
	if c.MaxFileSize <= 0 {
		return errors.New("max file size must be positive")
	}

	switch c.Auth {
	case AuthNone:
	case AuthBasic:
		if c.BasicUser == "" || c.BasicPass == "" {
			return errors.New("basic auth requires VELDOC_BASIC_USER and VELDOC_BASIC_PASSWORD")
		}
	case AuthForm:
		if c.FormUser == "" || c.FormPass == "" {
			return errors.New("form auth requires VELDOC_FORM_USER and VELDOC_FORM_PASSWORD")
		}
		if c.SessionSecret == "" {
			return errors.New("form auth requires VELDOC_SESSION_SECRET")
		}
	case AuthToken:
		if c.Token == "" {
			return errors.New("token auth requires VELDOC_TOKEN")
		}
	default:
		return fmt.Errorf("unsupported auth mode: %s", c.Auth)
	}

	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseSize(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("empty size")
	}

	multipliers := map[string]int64{
		"KiB": 1024,
		"MiB": 1024 * 1024,
		"GiB": 1024 * 1024 * 1024,
		"KB":  1000,
		"MB":  1000 * 1000,
		"GB":  1000 * 1000 * 1000,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(value, suffix) {
			num := strings.TrimSuffix(value, suffix)
			n, err := strconv.ParseInt(num, 10, 64)
			if err != nil {
				return 0, err
			}
			return n * mult, nil
		}
	}

	return strconv.ParseInt(value, 10, 64)
}
