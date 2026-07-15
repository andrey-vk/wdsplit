// Package model defines wdsplit's core domain types, mirroring the schema
// in internal/db/migrations.
package model

import "time"

type Category struct {
	ID   int64
	Name string
}

type Service struct {
	ID         int64
	CategoryID int64
	Name       string
}

type CatalogMode struct {
	ID   int64
	Name string
}

// AdapterKind distinguishes manual (GUI-edited, no source) adapters from
// every other adapter, which now all run identically through the JS
// sandbox regardless of upstream format — see internal/adapter/jsrun.
type AdapterKind string

const (
	AdapterManual AdapterKind = "manual"
	AdapterJS     AdapterKind = "js"
)

// Adapter is either a manual (GUI-edited) source or a sandboxed JS
// script. Builtin JS adapters are read-only; forking one for
// customization creates a new Adapter row with TemplateSourceID set,
// never edits the builtin in place.
type Adapter struct {
	ID   int64
	Name string
	Kind AdapterKind

	// SourceCode is nil for AdapterManual.
	SourceCode *string

	IsBuiltin bool

	// Version increments each time a builtin's embedded seed source
	// changes on upgrade. Meaningless for non-builtin adapters.
	Version int

	// Set only on a forked (non-builtin) adapter.
	TemplateSourceID      *int64
	TemplateVersionAtFork *int
}

// Feed is one (adapter, config) instance producing CatalogEntry rows. A nil
// OwnerRouterID means it's admin-owned and must be assigned to a
// CatalogMode to become visible; a set OwnerRouterID means it's a private
// manual feed belonging to exactly that router, always part of its catalog.
type Feed struct {
	ID            int64
	AdapterID     int64
	Name          string
	URL           *string
	OwnerRouterID *int64
	Enabled       bool
	LastSyncedAt  *time.Time
	LastSyncError *string
}

// EntryType is the kind of match a CatalogEntry represents.
type EntryType string

const (
	EntryDomain        EntryType = "domain"
	EntryDomainSuffix  EntryType = "domain_suffix"
	EntryDomainKeyword EntryType = "domain_keyword"
	EntryCIDR          EntryType = "cidr"
)

// CatalogEntry is a single domain/CIDR rule produced by a Feed and tagged
// to a Service. ServiceID lives here rather than on Feed because one feed
// can populate multiple services.
type CatalogEntry struct {
	ID        int64
	FeedID    int64
	ServiceID int64
	Type      EntryType
	Value     string
}

// Router is both the managed MikroTik router and its end user (User =
// Router, 1:1 — see project memory). Credentials are stored with
// app-level AES-256-GCM encryption; APIPasswordEncrypted/APIPasswordNonce
// hold the ciphertext and nonce, KeyVersion supports future key rotation.
type Router struct {
	ID                   int64
	Name                 string
	ActiveCatalogModeID  *int64
	APIHost              string
	APIPort              int
	APIUsername          string
	APIPasswordEncrypted []byte
	APIPasswordNonce     []byte
	KeyVersion           int
	WebAuthMode          string
	WebLogin             *string
	WebPasswordHash      *string
	CreatedAt            time.Time
}

// RouterSelection is one toggle: either CategoryID or ServiceID is set,
// never both, per the app-level rule that a category selection and
// per-service selections within it are mutually exclusive.
type RouterSelection struct {
	ID            int64
	RouterID      int64
	CatalogModeID int64
	CategoryID    *int64
	ServiceID     *int64
	Enabled       bool
}
