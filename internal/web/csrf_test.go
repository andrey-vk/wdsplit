package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidCSRFRequestSafeMethodsExempt(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		r := httptest.NewRequest(method, "/", nil)
		if !validCSRFRequest(r) {
			t.Errorf("validCSRFRequest() = false for %s, want true (safe methods are exempt)", method)
		}
	}
}

func TestValidCSRFRequestMatchingCookieAndHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "abc123"}) //nolint:gosec // request-side fixture, not a Set-Cookie response; Secure/HttpOnly/SameSite don't apply
	r.Header.Set(csrfHeaderName, "abc123")

	if !validCSRFRequest(r) {
		t.Error("validCSRFRequest() = false for matching cookie/header, want true")
	}
}

func TestValidCSRFRequestMismatch(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "abc123"}) //nolint:gosec // request-side fixture, not a Set-Cookie response; Secure/HttpOnly/SameSite don't apply
	r.Header.Set(csrfHeaderName, "different")

	if validCSRFRequest(r) {
		t.Error("validCSRFRequest() = true for mismatched cookie/header, want false")
	}
}

func TestValidCSRFRequestMissingCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set(csrfHeaderName, "abc123")

	if validCSRFRequest(r) {
		t.Error("validCSRFRequest() = true with no cookie, want false")
	}
}

func TestValidCSRFRequestMissingHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "abc123"}) //nolint:gosec // request-side fixture, not a Set-Cookie response; Secure/HttpOnly/SameSite don't apply

	if validCSRFRequest(r) {
		t.Error("validCSRFRequest() = true with no header, want false")
	}
}

func TestSetCSRFCookieProducesDistinctTokens(t *testing.T) {
	w1 := httptest.NewRecorder()
	setCSRFCookie(w1, false, 0, time.Time{})
	w2 := httptest.NewRecorder()
	setCSRFCookie(w2, false, 0, time.Time{})

	c1 := w1.Result().Cookies()[0].Value
	c2 := w2.Result().Cookies()[0].Value
	if c1 == c2 {
		t.Error("two setCSRFCookie calls produced the same token")
	}
}
