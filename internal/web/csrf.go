package web

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

// csrfCookieName/csrfHeaderName implement the double-submit-cookie
// pattern: the browser can read this cookie (deliberately not HttpOnly)
// and echo its value back as a request header, which a same-origin XHR
// can do but a cross-site form/script cannot — that's what stops the
// forgery. The cookie's presence alone proves nothing by itself.
const (
	csrfCookieName = "wdsplit_csrf"
	csrfHeaderName = "X-CSRF-Token"
)

func newCSRFToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// setCSRFCookie issues a new CSRF cookie, mirroring the admin session
// cookie's lifetime.
func setCSRFCookie(w http.ResponseWriter, secure bool, maxAge int, expires time.Time) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // deliberately not HttpOnly, see csrfCookieName doc; Secure computed at runtime
		Name:     csrfCookieName,
		Value:    newCSRFToken(),
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
		Expires:  expires,
	})
}

// clearCSRFCookie removes the CSRF cookie, mirroring logout.
func clearCSRFCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // deliberately not HttpOnly, see csrfCookieName doc; Secure computed at runtime
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// validCSRFRequest reports whether a mutating request carries a CSRF
// header matching its CSRF cookie. Safe methods never mutate state and
// are exempt.
func validCSRFRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}

	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	header := r.Header.Get(csrfHeaderName)
	if header == "" {
		return false
	}
	return hmac.Equal([]byte(header), []byte(cookie.Value))
}
