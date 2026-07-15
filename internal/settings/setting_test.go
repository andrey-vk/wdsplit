package settings

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	values  map[string]string
	saved   map[string]string
	saveErr error
}

func newFakeStore(values map[string]string) *fakeStore {
	if values == nil {
		values = map[string]string{}
	}
	return &fakeStore{values: values, saved: map[string]string{}}
}

func (f *fakeStore) GetAll(context.Context) (map[string]string, error) {
	return f.values, nil
}

func (f *fakeStore) Save(_ context.Context, key, value string) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved[key] = value
	return nil
}

func TestDefaultUsedWhenUnset(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		Parse:   ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := s.Get(); got != 42 {
		t.Errorf("Get() = %d, want 42 (default)", got)
	}
	if s.IsEnvLocked() {
		t.Error("IsEnvLocked() = true, want false")
	}
}

func TestDBOverridesDefault(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{"sync_interval": "100"}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		Parse:   ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := s.Get(); got != 100 {
		t.Errorf("Get() = %d, want 100 (db value)", got)
	}
}

func TestEnvOverridesDB(t *testing.T) {
	t.Setenv("WDSPLIT_SYNC_INTERVAL", "999")

	store := newFakeStore(nil)
	s, err := New(store, map[string]string{"sync_interval": "100"}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		EnvVar:  "WDSPLIT_SYNC_INTERVAL",
		Parse:   ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := s.Get(); got != 999 {
		t.Errorf("Get() = %d, want 999 (env value)", got)
	}
	if !s.IsEnvLocked() {
		t.Error("IsEnvLocked() = false, want true")
	}
}

func TestEnvOnlySettingCannotBeSet(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[string]{
		Default: "/data/wdsplit.sqlite3",
		EnvVar:  "WDSPLIT_DB",
		Parse:   ParseString,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Set(context.Background(), "/other/path.sqlite3"); !errors.Is(err, ErrEnvOnly) {
		t.Errorf("Set() error = %v, want ErrEnvOnly", err)
	}
}

func TestEnvLockedSettingCannotBeSet(t *testing.T) {
	t.Setenv("WDSPLIT_SYNC_INTERVAL", "999")

	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		EnvVar:  "WDSPLIT_SYNC_INTERVAL",
		Parse:   ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Set(context.Background(), "100"); err == nil {
		t.Error("Set() on env-locked setting = nil error, want error")
	}
}

func TestSetPersistsBeforeUpdatingCache(t *testing.T) {
	store := newFakeStore(nil)
	store.saveErr = errors.New("disk full")

	s, err := New(store, map[string]string{}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		Parse:   ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s.Set(context.Background(), "100"); err == nil {
		t.Fatal("Set() error = nil, want error from failed save")
	}
	if got := s.Get(); got != 42 {
		t.Errorf("Get() after failed Set = %d, want unchanged default 42", got)
	}
}

func TestSetUpdatesCacheAndFiresListeners(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		Parse:   ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var got int
	calls := 0
	s.OnChange(func(v int) {
		calls++
		got = v
	})

	if err := s.Set(context.Background(), "100"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if s.Get() != 100 {
		t.Errorf("Get() = %d, want 100", s.Get())
	}
	if calls != 1 {
		t.Errorf("listener called %d times, want 1", calls)
	}
	if got != 100 {
		t.Errorf("listener received %d, want 100", got)
	}
	if store.saved["sync_interval"] != "100" {
		t.Errorf("store.saved[sync_interval] = %q, want \"100\"", store.saved["sync_interval"])
	}
}

func TestOnChangeUnsubscribe(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		Parse:   ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	calls := 0
	unsubscribe := s.OnChange(func(int) { calls++ })
	unsubscribe()

	if err := s.Set(context.Background(), "100"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if calls != 0 {
		t.Errorf("listener called %d times after unsubscribe, want 0", calls)
	}
}

func TestInvalidDBValueFallsBackToDefault(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{"sync_interval": "not-a-number"}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		Parse:   ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New() error = %v, want nil (should fall back, not fail)", err)
	}
	if got := s.Get(); got != 42 {
		t.Errorf("Get() = %d, want 42 (default fallback)", got)
	}
}

func TestInvalidEnvValueFailsConstruction(t *testing.T) {
	t.Setenv("WDSPLIT_SYNC_INTERVAL", "not-a-number")

	store := newFakeStore(nil)
	_, err := New(store, map[string]string{}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		EnvVar:  "WDSPLIT_SYNC_INTERVAL",
		Parse:   ParseInt,
	}, nil)
	if err == nil {
		t.Error("New() error = nil, want error for invalid env value")
	}
}

func TestValidateRejectsOutOfRangeValue(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[int]{
		Default: 42,
		DBKey:   "sync_interval",
		Parse:   ParseInt,
		Validate: func(v int) error {
			if v < 1 {
				return errors.New("must be positive")
			}
			return nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Set(context.Background(), "-5"); err == nil {
		t.Error("Set(-5) error = nil, want validation error")
	}
	if got := s.Get(); got != 42 {
		t.Errorf("Get() after rejected Set = %d, want unchanged default 42", got)
	}
}

func TestValidateDoesNotApply(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[int]{
		Key: "sync_interval", Default: 42, DBKey: "sync_interval", Parse: ParseInt,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s.Validate("100"); err != nil {
		t.Fatalf("Validate(100): %v", err)
	}
	if got := s.Get(); got != 42 {
		t.Errorf("Get() after Validate() = %d, want unchanged default 42 (Validate must not apply)", got)
	}
	if len(store.saved) != 0 {
		t.Errorf("store.saved = %v, want empty (Validate must not persist)", store.saved)
	}
}

func TestValidateRejectsEnvOnlyAndEnvLocked(t *testing.T) {
	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[string]{
		Key: "db_path", EnvVar: "WDSPLIT_DB", Parse: ParseString,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Validate("whatever"); !errors.Is(err, ErrEnvOnly) {
		t.Errorf("Validate() error = %v, want ErrEnvOnly", err)
	}
}

func TestInfoAccessors(t *testing.T) {
	store := newFakeStore(map[string]string{"admin_cookie_secure": "false"})
	s, err := New(store, map[string]string{"admin_cookie_secure": "false"}, Spec[string]{
		Key:     "admin_cookie_secure",
		Default: "auto",
		DBKey:   "admin_cookie_secure",
		EnvVar:  "WDSPLIT_ADMIN_COOKIE_SECURE",
		UIType:  "select",
		Options: []string{"auto", "true", "false"},
		Parse:   ParseString,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var info Info = s
	if got := info.Key(); got != "admin_cookie_secure" {
		t.Errorf("Key() = %q, want %q", got, "admin_cookie_secure")
	}
	if got := info.EnvVar(); got != "WDSPLIT_ADMIN_COOKIE_SECURE" {
		t.Errorf("EnvVar() = %q, want %q", got, "WDSPLIT_ADMIN_COOKIE_SECURE")
	}
	if info.IsEnvLocked() {
		t.Error("IsEnvLocked() = true, want false (value came from DB, not env)")
	}
	if info.IsSecret() {
		t.Error("IsSecret() = true, want false")
	}
	if got := info.UIType(); got != "select" {
		t.Errorf("UIType() = %q, want %q", got, "select")
	}
	if got := info.Options(); len(got) != 3 {
		t.Errorf("Options() = %v, want 3 entries", got)
	}
	if got := info.ValueString(); got != "false" {
		t.Errorf("ValueString() = %q, want %q", got, "false")
	}
	if got := info.DefaultString(); got != "auto" {
		t.Errorf("DefaultString() = %q, want %q", got, "auto")
	}
}

func TestSecretSettingHidesValue(t *testing.T) {
	t.Setenv("WDSPLIT_ADMIN_PASSWORD", "hunter2")

	store := newFakeStore(nil)
	s, err := New(store, map[string]string{}, Spec[string]{
		Key: "admin_password", EnvVar: "WDSPLIT_ADMIN_PASSWORD", Secret: true, UIType: "password", Parse: ParseString,
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var info Info = s
	if got := info.ValueString(); got != "" {
		t.Errorf("ValueString() = %q, want empty for a secret", got)
	}
	if got := info.DefaultString(); got != "" {
		t.Errorf("DefaultString() = %q, want empty for a secret", got)
	}
	if !info.IsSecret() {
		t.Error("IsSecret() = false, want true")
	}
	// The actual value is still functionally set and usable — only its
	// string representation is withheld.
	if s.Get() != "hunter2" {
		t.Error("Get() should still return the real value internally")
	}
}
