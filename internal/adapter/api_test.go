package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestAPIRunner builds an apiRunner with a plain, unhardened
// http.Client so tests can point it at an httptest.Server, which listens
// on loopback — a target the production SSRF-hardened client (see
// http.go) deliberately refuses to connect to. This only exercises the
// request-counting/byte-limit logic in doGet, not the SSRF dialer itself
// (that's tested separately in http_test.go, and end-to-end against a
// real public host in adapter_live_test.go).
func newTestAPIRunner(limits Limits) *apiRunner {
	return &apiRunner{
		ctx:      context.Background(),
		limits:   limits,
		client:   http.DefaultClient,
		feedName: "test",
	}
}

func writeTestBody(t testing.TB, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write test response body: %v", err)
	}
}

func TestDoGetReturnsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestBody(t, w, "hello world")
	}))
	defer srv.Close()

	r := newTestAPIRunner(testLimits())
	body, err := r.doGet(srv.URL)
	if err != nil {
		t.Fatalf("doGet: %v", err)
	}
	if body != "hello world" {
		t.Errorf("body = %q, want %q", body, "hello world")
	}
}

func TestDoGetNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	r := newTestAPIRunner(testLimits())
	if _, err := r.doGet(srv.URL); err == nil {
		t.Fatal("doGet() error = nil, want error for HTTP 500")
	}
}

func TestDoGetMaxRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestBody(t, w, "ok")
	}))
	defer srv.Close()

	limits := testLimits()
	limits.MaxRequests = 2
	r := newTestAPIRunner(limits)

	for i := range 2 {
		if _, err := r.doGet(srv.URL); err != nil {
			t.Fatalf("doGet() call %d: %v", i+1, err)
		}
	}
	if _, err := r.doGet(srv.URL); err == nil {
		t.Fatal("doGet() call 3 error = nil, want error (exceeds max requests)")
	}
}

func TestDoGetMaxResponseBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestBody(t, w, strings.Repeat("x", 1000))
	}))
	defer srv.Close()

	limits := testLimits()
	limits.MaxResponseBytes = 100
	r := newTestAPIRunner(limits)

	if _, err := r.doGet(srv.URL); err == nil {
		t.Fatal("doGet() error = nil, want error (response exceeds max response bytes)")
	}
}

func TestDoGetMaxTotalBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestBody(t, w, strings.Repeat("x", 60))
	}))
	defer srv.Close()

	limits := testLimits()
	limits.MaxResponseBytes = 1000 // each individual response is fine
	limits.MaxTotalBytes = 100     // but two of them together aren't
	limits.MaxRequests = 5
	r := newTestAPIRunner(limits)

	if _, err := r.doGet(srv.URL); err != nil {
		t.Fatalf("doGet() call 1: %v", err)
	}
	if _, err := r.doGet(srv.URL); err == nil {
		t.Fatal("doGet() call 2 error = nil, want error (exceeds max total bytes)")
	}
}

func TestDoGetRejectsNonHTTPScheme(t *testing.T) {
	r := newTestAPIRunner(testLimits())
	if _, err := r.doGet("file:///etc/passwd"); err == nil {
		t.Fatal("doGet() error = nil, want error (non-http scheme rejected)")
	}
}

func TestDoGetHostAllowlist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestBody(t, w, "ok")
	}))
	defer srv.Close()

	limits := testLimits()
	limits.AllowedHosts = []string{"totally-different-host.invalid"}
	r := newTestAPIRunner(limits)

	if _, err := r.doGet(srv.URL); err == nil {
		t.Fatal("doGet() error = nil, want error (host not in allowlist)")
	}
}

func TestHTTPGetThroughScriptEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestBody(t, w, "full.example.com\ndomain.example.com\n")
	}))
	defer srv.Close()

	// Exercises Run() end-to-end but with a directly-constructed
	// apiRunner substituted in place of the one Run() would normally
	// build, to avoid the SSRF-hardened client rejecting the loopback
	// test server. This still proves the JS <-> Go httpGet binding
	// itself works, just not the SSRF dialer (tested separately).
	source := `
function sync(feed, api) {
    var text = api.httpGet(feed.url);
    var lines = text.trim().split("\n");
    var filters = lines.map(function(l) { return {type: "domain_suffix", value: l}; });
    return [{category: "Test", service: "Svc", filters: filters}];
}
`
	groups, err := runWithClient(context.Background(), source, Feed{URL: srv.URL}, testLimits(), http.DefaultClient)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(groups) != 1 || len(groups[0].Entries) != 2 {
		t.Fatalf("unexpected result: %+v", groups)
	}
}
