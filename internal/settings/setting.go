// Package settings implements runtime-configurable application settings
// with a fixed precedence: env var > stored DB value > hardcoded default.
//
// Each Setting caches its resolved value; consumers call Get every time
// they need it rather than holding onto a snapshot. Env precedence is
// resolved once, at construction — changing an env var after the process
// starts has no effect until restart. Set persists to the store before
// mutating the cache, so a failed write never leaves Get returning a value
// that isn't actually durable; on success it then notifies OnChange
// listeners synchronously.
package settings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// ErrEnvOnly is returned by Set when the setting has no DBKey and can only
// ever be supplied via its env var.
var ErrEnvOnly = errors.New("setting is env-only and cannot be changed")

// OnChangeFunc is called with a setting's new value after Set applies it.
type OnChangeFunc[T any] func(newValue T)

// Spec describes one setting: its default, where it may be sourced from,
// how to parse/validate a raw string value, and metadata an admin UI
// needs to render and list it.
type Spec[T any] struct {
	// Key is the stable identifier used for API/UI purposes — always
	// required, independent of DBKey, since env-only settings have no
	// DBKey but still need an identifier to be listed.
	Key string

	Default T

	// DBKey is the row key in the setting table. Empty means the setting
	// is env-only and can never be persisted or changed via Set.
	DBKey string

	// EnvVar is the environment variable that overrides the DB value and
	// the default. Empty means there is no env override.
	EnvVar string

	// Secret means the value is never exposed via Info.ValueString (e.g.
	// a password) — only whether it's set.
	Secret bool

	// UIType hints how an admin UI should render this setting: "string",
	// "password", "int", "bool", or "select" (with Options).
	UIType string

	// Options lists the valid values for UIType "select".
	Options []string

	Parse func(string) (T, error)

	// Validate is optional; nil means any value Parse accepts is valid.
	Validate func(T) error
}

// Info is the non-generic view of a Setting an admin UI/API can list and
// update without knowing its underlying Go type — Validate and Set both
// take a raw string, so unlike Get they don't depend on T and can be
// expressed on a common interface.
type Info interface {
	Key() string
	EnvVar() string
	IsEnvLocked() bool

	// Editable reports whether this setting has a DBKey at all — false
	// means env-only, permanently unsettable via Set, regardless of
	// whether the env var happens to be set right now. IsEnvLocked is a
	// separate, temporary condition: a DB-editable setting becomes
	// env-locked only while its env var is set, and would accept Set
	// again if that env var were removed and the process restarted.
	Editable() bool

	IsSecret() bool
	UIType() string
	Options() []string

	// ValueString/DefaultString are empty for secrets.
	ValueString() string
	DefaultString() string

	// Validate parses and validates raw without applying it.
	Validate(raw string) error
	Set(ctx context.Context, raw string) error
}

// Setting is one runtime-configurable value.
type Setting[T any] struct {
	mu    sync.RWMutex
	value T

	key       string
	def       T
	dbKey     string
	envVar    string
	envLocked bool
	secret    bool
	uiType    string
	options   []string

	parse    func(string) (T, error)
	validate func(T) error
	store    Store

	listenersMu sync.Mutex
	listeners   []OnChangeFunc[T]
}

var _ Info = (*Setting[string])(nil)

// New resolves a setting's effective value from dbValues (as returned by
// Store.GetAll, loaded once by the caller) and the environment, in that
// precedence order, and returns a Setting caching the result.
//
// An invalid env value is a hard error, since it means deployment
// configuration is broken. An invalid stored DB value instead falls back
// to the default and logs a warning, so a bad row can't block startup.
func New[T any](store Store, dbValues map[string]string, spec Spec[T], logger *slog.Logger) (*Setting[T], error) {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Setting[T]{
		key:      spec.Key,
		def:      spec.Default,
		dbKey:    spec.DBKey,
		envVar:   spec.EnvVar,
		secret:   spec.Secret,
		uiType:   spec.UIType,
		options:  spec.Options,
		parse:    spec.Parse,
		validate: spec.Validate,
		store:    store,
	}

	raw, source, ok := resolve(dbValues, spec)
	if !ok {
		s.value = spec.Default
		return s, nil
	}

	v, err := spec.Parse(raw)
	if err == nil && spec.Validate != nil {
		err = spec.Validate(v)
	}
	if err != nil {
		if source == sourceEnv {
			return nil, fmt.Errorf("invalid value for env var %s: %w", spec.EnvVar, err)
		}
		logger.Warn("invalid stored setting value, using default", "key", spec.DBKey, "error", err)
		s.value = spec.Default
		return s, nil
	}

	s.value = v
	s.envLocked = source == sourceEnv
	return s, nil
}

type source int

const (
	sourceDefault source = iota
	sourceDB
	sourceEnv
)

func resolve[T any](dbValues map[string]string, spec Spec[T]) (raw string, src source, ok bool) {
	if spec.EnvVar != "" {
		if v := os.Getenv(spec.EnvVar); v != "" {
			return v, sourceEnv, true
		}
	}
	if spec.DBKey != "" {
		if v, present := dbValues[spec.DBKey]; present {
			return v, sourceDB, true
		}
	}
	return "", sourceDefault, false
}

// Get returns the current cached value.
func (s *Setting[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// IsEnvLocked reports whether this setting's value came from its env var,
// resolved once at construction time.
func (s *Setting[T]) IsEnvLocked() bool {
	return s.envLocked
}

// Validate parses and validates raw without applying it — used to check a
// batch of updates before committing any of them (see Set).
func (s *Setting[T]) Validate(raw string) error {
	if s.dbKey == "" {
		return ErrEnvOnly
	}
	if s.envLocked {
		return fmt.Errorf("setting is overridden by env var %s and cannot be changed", s.envVar)
	}
	_, err := s.parseAndValidate(raw)
	return err
}

func (s *Setting[T]) parseAndValidate(raw string) (T, error) {
	v, err := s.parse(raw)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("invalid value: %w", err)
	}
	if s.validate != nil {
		if err := s.validate(v); err != nil {
			var zero T
			return zero, err
		}
	}
	return v, nil
}

// Set parses and validates raw, persists it via the store, updates the
// cached value, and notifies OnChange listeners — in that order, so a
// failed persist never changes what Get returns.
func (s *Setting[T]) Set(ctx context.Context, raw string) error {
	if s.dbKey == "" {
		return ErrEnvOnly
	}
	if s.envLocked {
		return fmt.Errorf("setting is overridden by env var %s and cannot be changed", s.envVar)
	}

	v, err := s.parseAndValidate(raw)
	if err != nil {
		return err
	}

	if err := s.store.Save(ctx, s.dbKey, raw); err != nil {
		return fmt.Errorf("save setting: %w", err)
	}

	s.mu.Lock()
	s.value = v
	s.mu.Unlock()

	s.fireCallbacks(v)
	return nil
}

// Key returns the setting's stable API identifier.
func (s *Setting[T]) Key() string { return s.key }

// EnvVar returns the env var that can override this setting, or "".
func (s *Setting[T]) EnvVar() string { return s.envVar }

// Editable reports whether this setting has a DBKey — see the Info
// interface doc for how this differs from IsEnvLocked.
func (s *Setting[T]) Editable() bool { return s.dbKey != "" }

// IsSecret reports whether ValueString/DefaultString withhold the value.
func (s *Setting[T]) IsSecret() bool { return s.secret }

// UIType hints how an admin UI should render this setting.
func (s *Setting[T]) UIType() string { return s.uiType }

// Options lists the valid values for UIType "select".
func (s *Setting[T]) Options() []string { return s.options }

// ValueString returns the current value formatted as a string, or "" if
// this setting is secret.
func (s *Setting[T]) ValueString() string {
	if s.secret {
		return ""
	}
	return fmt.Sprintf("%v", s.Get())
}

// DefaultString returns the default value formatted as a string, or "" if
// this setting is secret.
func (s *Setting[T]) DefaultString() string {
	if s.secret {
		return ""
	}
	return fmt.Sprintf("%v", s.def)
}

// OnChange registers fn to be called after every successful Set. It
// returns an unsubscribe function.
func (s *Setting[T]) OnChange(fn OnChangeFunc[T]) (unsubscribe func()) {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()

	idx := len(s.listeners)
	s.listeners = append(s.listeners, fn)

	return func() {
		s.listenersMu.Lock()
		defer s.listenersMu.Unlock()
		s.listeners[idx] = nil
	}
}

func (s *Setting[T]) fireCallbacks(v T) {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()
	for _, fn := range s.listeners {
		if fn != nil {
			fn(v)
		}
	}
}
