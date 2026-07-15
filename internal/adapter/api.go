package adapter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/dop251/goja"
)

// apiRunner backs the `api` object passed to an adapter script's
// sync(feed, api). Every method that can fail panics with a goja error
// rather than returning one, since Go functions registered directly on a
// goja object can't return (value, error) and have goja translate the
// error into a catchable JS exception — panicking with vm.NewGoError does
// that translation instead, matching goja's documented pattern for this.
type apiRunner struct {
	ctx      context.Context
	vm       *goja.Runtime
	limits   Limits
	client   *http.Client
	feedName string

	requests   int
	totalBytes int
}

func (r *apiRunner) httpGet(rawURL string) string {
	body, err := r.doGet(rawURL)
	if err != nil {
		panic(r.vm.NewGoError(err))
	}
	return body
}

func (r *apiRunner) log(msg string) {
	slog.Info("adapter log", "feed", r.feedName, "message", msg)
}

func (r *apiRunner) doGet(rawURL string) (string, error) {
	if r.requests >= r.limits.MaxRequests {
		return "", fmt.Errorf("exceeded max requests (%d)", r.limits.MaxRequests)
	}
	r.requests++

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if len(r.limits.AllowedHosts) > 0 && !hostAllowed(u.Hostname(), r.limits.AllowedHosts) {
		return "", fmt.Errorf("host %q is not allowed", u.Hostname())
	}

	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Warn("close response body", "url", rawURL, "error", cerr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch %s: HTTP %s", rawURL, resp.Status)
	}

	limited := io.LimitReader(resp.Body, int64(r.limits.MaxResponseBytes)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read response from %s: %w", rawURL, err)
	}
	if len(body) > r.limits.MaxResponseBytes {
		return "", fmt.Errorf("response from %s exceeds max response bytes (%d)", rawURL, r.limits.MaxResponseBytes)
	}

	r.totalBytes += len(body)
	if r.totalBytes > r.limits.MaxTotalBytes {
		return "", fmt.Errorf("exceeded max total bytes (%d) across all requests", r.limits.MaxTotalBytes)
	}

	return string(body), nil
}
