// Package adapter runs adapter scripts in a sandboxed goja JS runtime.
// Every non-manual adapter (builtin or user-forked) executes identically
// through this one path — parsing logic lives entirely in the script, not
// in Go. See project memory for the design discussion.
package adapter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dop251/goja"

	"github.com/andrey-vk/wdsplit/internal/model"
)

// Entry is one parsed rule within a ServiceGroup.
type Entry struct {
	Type  model.EntryType
	Value string
}

// ServiceGroup is one {category, service, filters} group returned by an
// adapter script's sync(feed, api) function.
type ServiceGroup struct {
	Category string
	Service  string
	Entries  []Entry
}

// Feed is passed to the script as the `feed` parameter: {id, name, url,
// data}. Data is the feed's raw JSON configuration string — the script
// parses it itself with JSON.parse(feed.data) if it needs it.
type Feed struct {
	ID   int64
	Name string
	URL  string
	Data string
}

// Limits bounds what a script run may do. All fields are required; there
// are no implicit defaults, since silently running unbounded is the
// failure mode this whole package exists to prevent.
type Limits struct {
	Timeout          time.Duration
	MaxSourceBytes   int
	MaxResponseBytes int
	MaxTotalBytes    int
	MaxRequests      int
	MaxEntries       int
	MaxCallStack     int

	// AllowedHosts restricts api.httpGet to these hosts. Empty means any
	// host is allowed (still subject to the always-on public-IP check —
	// see http.go). Hostname comparison is case-insensitive, exact match.
	AllowedHosts []string
}

// jsFilter/jsGroup mirror the shape a script's sync() must return:
// [{category, service, filters: [{type, value}]}]. Field names are
// lowercase to match plain JS object literals (see the TagFieldNameMapper
// call in Run).
type jsFilter struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type jsGroup struct {
	Category string     `json:"category"`
	Service  string     `json:"service"`
	Filters  []jsFilter `json:"filters"`
}

// Run compiles and executes source, calling its sync(feed, api) function,
// and returns the parsed groups. A script that throws, exceeds a limit,
// or returns a malformed shape produces an error and no partial results —
// callers must not apply anything from a failed run.
func Run(ctx context.Context, source string, feed Feed, limits Limits) ([]ServiceGroup, error) {
	return runWithClient(ctx, source, feed, limits, newSandboxHTTPClient(limits.AllowedHosts))
}

// runWithClient is Run with the HTTP client injectable, so tests can
// substitute an unhardened client pointed at an httptest.Server (which
// listens on loopback — a target the production SSRF-hardened client
// deliberately refuses). Production code should only ever call Run.
func runWithClient(ctx context.Context, source string, feed Feed, limits Limits, client *http.Client) ([]ServiceGroup, error) {
	if len(source) > limits.MaxSourceBytes {
		return nil, fmt.Errorf("adapter source is %d bytes, exceeds max of %d", len(source), limits.MaxSourceBytes)
	}

	runCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	vm.SetMaxCallStackSize(limits.MaxCallStack)

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-runCtx.Done():
			vm.Interrupt(runCtx.Err())
		case <-done:
		}
	}()

	if _, err := vm.RunString(source); err != nil {
		return nil, fmt.Errorf("evaluate adapter source: %w", err)
	}

	syncFn, ok := goja.AssertFunction(vm.Get("sync"))
	if !ok {
		return nil, errors.New("adapter must define function sync(feed, api)")
	}

	runner := &apiRunner{ctx: runCtx, vm: vm, limits: limits, client: client, feedName: feed.Name}

	feedVal := vm.ToValue(map[string]any{
		"id":   feed.ID,
		"name": feed.Name,
		"url":  feed.URL,
		"data": feed.Data,
	})

	apiVal := vm.NewObject()
	if err := apiVal.Set("httpGet", runner.httpGet); err != nil {
		return nil, fmt.Errorf("build api object: %w", err)
	}
	if err := apiVal.Set("log", runner.log); err != nil {
		return nil, fmt.Errorf("build api object: %w", err)
	}

	result, err := syncFn(goja.Undefined(), feedVal, apiVal)
	if err != nil {
		return nil, fmt.Errorf("run adapter: %w", err)
	}

	var raw []jsGroup
	if err := vm.ExportTo(result, &raw); err != nil {
		return nil, fmt.Errorf("adapter returned an unexpected shape: %w", err)
	}

	return toServiceGroups(raw, limits.MaxEntries)
}

func toServiceGroups(raw []jsGroup, maxEntries int) ([]ServiceGroup, error) {
	groups := make([]ServiceGroup, 0, len(raw))
	total := 0

	for _, g := range raw {
		if g.Category == "" || g.Service == "" {
			return nil, errors.New("adapter returned a group with an empty category or service")
		}

		entries := make([]Entry, 0, len(g.Filters))
		for _, f := range g.Filters {
			total++
			if total > maxEntries {
				return nil, fmt.Errorf("adapter produced more than the max of %d entries", maxEntries)
			}

			entryType, ok := validEntryType(f.Type)
			if !ok {
				return nil, fmt.Errorf("invalid entry type %q", f.Type)
			}
			if f.Value == "" {
				return nil, fmt.Errorf("entry of type %q has an empty value", f.Type)
			}

			entries = append(entries, Entry{Type: entryType, Value: f.Value})
		}

		groups = append(groups, ServiceGroup{Category: g.Category, Service: g.Service, Entries: entries})
	}

	return groups, nil
}

func validEntryType(s string) (model.EntryType, bool) {
	switch model.EntryType(s) {
	case model.EntryDomain, model.EntryDomainSuffix, model.EntryDomainKeyword, model.EntryCIDR:
		return model.EntryType(s), true
	default:
		return "", false
	}
}
