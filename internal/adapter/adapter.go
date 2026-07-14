// Package adapter defines the shared shape every upstream rule-set format
// parser implements, plus a common HTTP fetch helper. Concrete adapters
// live in subpackages (e.g. internal/adapter/v2fly).
package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/andrey-vk/wdsplit/internal/model"
)

// Entry is a single parsed rule extracted from an upstream feed, not yet
// tied to a database row (a Feed sync assigns FeedID/ServiceID afterward).
type Entry struct {
	Type  model.EntryType
	Value string
}

// Parser turns raw bytes from one upstream source into Entries. Kept
// separate from fetching so parsing logic is testable without network
// access.
type Parser interface {
	Parse(r io.Reader) ([]Entry, error)
}

// Fetch retrieves url's body via HTTP GET for a Parser to consume. The
// caller must close the returned body. Shared by every adapter kind that
// fetches over HTTP (all current kinds except "manual").
func Fetch(ctx context.Context, client *http.Client, url string) (io.ReadCloser, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", url, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("fetch %s: unexpected status %s", url, resp.Status)
		if cerr := resp.Body.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close response body: %w", cerr))
		}
		return nil, err
	}

	return resp.Body, nil
}
