package config

import (
	"context"
	"testing"
)

type fakeStore struct {
	values map[string]string
}

func (f *fakeStore) GetAll(context.Context) (map[string]string, error) {
	return f.values, nil
}

func (f *fakeStore) Save(_ context.Context, key, value string) error {
	f.values[key] = value
	return nil
}

func TestLoadRequiresAdminPassword(t *testing.T) {
	t.Setenv("WDSPLIT_SESSION_SECRET", "s3cr3t")

	_, err := Load(context.Background(), &fakeStore{values: map[string]string{}}, nil)
	if err == nil {
		t.Error("Load() error = nil, want error (WDSPLIT_ADMIN_PASSWORD unset)")
	}
}

func TestLoadRequiresSessionSecret(t *testing.T) {
	t.Setenv("WDSPLIT_ADMIN_PASSWORD", "hunter2")

	_, err := Load(context.Background(), &fakeStore{values: map[string]string{}}, nil)
	if err == nil {
		t.Error("Load() error = nil, want error (WDSPLIT_SESSION_SECRET unset)")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("WDSPLIT_ADMIN_PASSWORD", "hunter2")
	t.Setenv("WDSPLIT_SESSION_SECRET", "s3cr3t")

	cfg, err := Load(context.Background(), &fakeStore{values: map[string]string{}}, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.AdminPassword.Get(); got != "hunter2" {
		t.Errorf("AdminPassword = %q, want %q", got, "hunter2")
	}
	if got := cfg.SessionMaxAge.Get(); got != 28800 {
		t.Errorf("SessionMaxAge default = %d, want 28800", got)
	}
	if got := cfg.AdminCookieSecure.Get(); got != "auto" {
		t.Errorf("AdminCookieSecure default = %q, want %q", got, "auto")
	}
}

func TestLoadRejectsInvalidAdminCookieSecure(t *testing.T) {
	t.Setenv("WDSPLIT_ADMIN_PASSWORD", "hunter2")
	t.Setenv("WDSPLIT_SESSION_SECRET", "s3cr3t")

	cfg, err := Load(context.Background(), &fakeStore{values: map[string]string{
		"admin_cookie_secure": "not-a-valid-value",
	}}, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Invalid stored values fall back to default rather than failing
	// startup — see internal/settings.New.
	if got := cfg.AdminCookieSecure.Get(); got != "auto" {
		t.Errorf("AdminCookieSecure = %q, want fallback to default %q", got, "auto")
	}
}
