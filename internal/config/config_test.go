package config

import (
	"testing"
)

func TestValidateAuthModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "none",
			cfg: Config{Root: "/data", Addr: ":8080", Auth: AuthNone, MaxFileSize: 1024},
		},
		{
			name: "basic missing creds",
			cfg:  Config{Root: "/data", Addr: ":8080", Auth: AuthBasic, MaxFileSize: 1024},
			wantErr: true,
		},
		{
			name: "basic ok",
			cfg: Config{
				Root: "/data", Addr: ":8080", Auth: AuthBasic,
				BasicUser: "u", BasicPass: "p", MaxFileSize: 1024,
			},
		},
		{
			name: "form missing secret",
			cfg: Config{
				Root: "/data", Addr: ":8080", Auth: AuthForm,
				FormUser: "u", FormPass: "p", MaxFileSize: 1024,
			},
			wantErr: true,
		},
		{
			name: "token missing",
			cfg:  Config{Root: "/data", Addr: ":8080", Auth: AuthToken, MaxFileSize: 1024},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	t.Parallel()

	size, err := parseSize("2MiB")
	if err != nil {
		t.Fatal(err)
	}
	if size != 2*1024*1024 {
		t.Fatalf("got %d", size)
	}
}
