// Package config defines wdsplit's concrete application settings, built on
// the generic engine in internal/settings.
package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/andrey-vk/wdsplit/internal/settings"
)

// Settings holds every application setting wdsplit currently has. Add
// fields here as features need them, not preemptively.
type Settings struct {
	// AdminPassword authenticates the single admin login. Env-only —
	// never stored in the database, matching the pattern of not
	// persisting credential material anywhere it doesn't have to be.
	AdminPassword *settings.Setting[string]

	// SessionSecret signs admin session/CSRF tokens. Env-only.
	SessionSecret *settings.Setting[string]

	// SessionMaxAge is the admin session lifetime in seconds. 0 means
	// session cookies (expire when the browser closes).
	SessionMaxAge *settings.Setting[int]

	// AdminCookieSecure controls the Secure flag on admin cookies:
	// "auto" (Secure iff the request arrived over TLS), "true", or
	// "false" (needed for local HTTP development).
	AdminCookieSecure *settings.Setting[string]
}

// Load resolves every setting from the store (once) and the environment.
func Load(ctx context.Context, store settings.Store, logger *slog.Logger) (*Settings, error) {
	dbValues, err := store.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}

	adminPassword, err := settings.New(store, dbValues, settings.Spec[string]{
		EnvVar: "WDSPLIT_ADMIN_PASSWORD",
		Parse:  settings.ParseString,
	}, logger)
	if err != nil {
		return nil, err
	}
	if adminPassword.Get() == "" {
		return nil, errors.New("WDSPLIT_ADMIN_PASSWORD is required")
	}

	sessionSecret, err := settings.New(store, dbValues, settings.Spec[string]{
		EnvVar: "WDSPLIT_SESSION_SECRET",
		Parse:  settings.ParseString,
	}, logger)
	if err != nil {
		return nil, err
	}
	if sessionSecret.Get() == "" {
		return nil, errors.New("WDSPLIT_SESSION_SECRET is required")
	}

	sessionMaxAge, err := settings.New(store, dbValues, settings.Spec[int]{
		Default: 28800, // 8 hours
		DBKey:   "session_max_age",
		EnvVar:  "WDSPLIT_SESSION_MAX_AGE",
		Parse:   settings.ParseInt,
		Validate: func(v int) error {
			if v < 0 {
				return errors.New("must be >= 0")
			}
			return nil
		},
	}, logger)
	if err != nil {
		return nil, err
	}

	adminCookieSecure, err := settings.New(store, dbValues, settings.Spec[string]{
		Default: "auto",
		DBKey:   "admin_cookie_secure",
		EnvVar:  "WDSPLIT_ADMIN_COOKIE_SECURE",
		Parse:   settings.ParseString,
		Validate: func(v string) error {
			switch v {
			case "auto", "true", "false":
				return nil
			}
			return fmt.Errorf("must be auto, true, or false, got %q", v)
		},
	}, logger)
	if err != nil {
		return nil, err
	}

	return &Settings{
		AdminPassword:     adminPassword,
		SessionSecret:     sessionSecret,
		SessionMaxAge:     sessionMaxAge,
		AdminCookieSecure: adminCookieSecure,
	}, nil
}
