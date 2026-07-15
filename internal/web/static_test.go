package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func testDistFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte("<html>index</html>")},
		"assets/app.js": {Data: []byte("console.log('app')")},
		"favicon.ico":   {Data: []byte("icon-bytes")},
	}
}

func TestSPAHandlerServesRealFile(t *testing.T) {
	h := spaHandler(testDistFS())

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != "console.log('app')" {
		t.Errorf("body = %q, want the real asset content", got)
	}
}

func TestSPAHandlerServesIndexAtRoot(t *testing.T) {
	h := spaHandler(testDistFS())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != "<html>index</html>" {
		t.Errorf("body = %q, want index.html content", got)
	}
}

func TestSPAHandlerFallsBackToIndexForClientRoute(t *testing.T) {
	h := spaHandler(testDistFS())

	// /login isn't a real file in the embedded build — it's a Vue Router
	// route, resolved client-side. The server must still return the SPA
	// shell (index.html), not a 404, so a direct link or page refresh
	// works.
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != "<html>index</html>" {
		t.Errorf("body = %q, want index.html content (SPA fallback)", got)
	}
}

func TestSPAHandlerFallsBackForNestedClientRoute(t *testing.T) {
	h := spaHandler(testDistFS())

	req := httptest.NewRequest(http.MethodGet, "/some/deep/client/route", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != "<html>index</html>" {
		t.Errorf("body = %q, want index.html content (SPA fallback)", got)
	}
}
