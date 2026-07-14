package web

import (
	"crypto/hmac"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// apiResponse is a standard JSON response envelope.
type apiResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}

// handleLogin handles POST /api/admin/login.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.loginLimiter.allow(s.clientIP(r)) {
		writeJSON(w, http.StatusTooManyRequests, apiResponse{OK: false, Error: "Rate limit exceeded"})
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "Invalid request body"})
		return
	}

	if !hmac.Equal([]byte(body.Password), []byte(s.settings.AdminPassword.Get())) {
		writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "Invalid password"})
		return
	}

	s.issueSession(w, r)
	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request) {
	maxAge := s.settings.SessionMaxAge.Get()
	var expires time.Time
	if maxAge > 0 {
		expires = time.Now().Add(time.Duration(maxAge) * time.Second)
	}

	secure := s.cookieSecure(r)
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure is set below, computed at runtime; gosec can't see through the variable
		Name:     adminCookieName,
		Value:    sessionToken(s.settings.SessionSecret.Get()),
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
		Expires:  expires,
	})
	setCSRFCookie(w, secure, maxAge, expires)
}

// handleMe handles GET /api/admin/me, the SPA's session-bootstrap call.
func (s *Server) handleMe(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true})
}

// handleLogout handles POST /api/admin/logout.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	secure := s.cookieSecure(r)
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure is set below, computed at runtime; gosec can't see through the variable
		Name:     adminCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	clearCSRFCookie(w, secure)
	writeJSON(w, http.StatusOK, apiResponse{OK: true})
}

// requireAdmin validates the admin session cookie and CSRF header on
// mutating requests, renews the session if it's aging but not yet
// expired, and self-heals a missing CSRF cookie so the next request from
// the same browser has a token to echo back.
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxAge := time.Duration(s.settings.SessionMaxAge.Get()) * time.Second
		if maxAge <= 0 {
			maxAge = 8 * time.Hour
		}

		cookie, err := r.Cookie(adminCookieName)
		if err != nil || !validSession(s.settings.SessionSecret.Get(), cookie.Value, maxAge) {
			writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
			return
		}

		if age, ok := sessionAge(cookie.Value); ok && age > time.Hour && age < maxAge-time.Hour {
			s.issueSession(w, r)
		}

		if _, err := r.Cookie(csrfCookieName); err != nil {
			setCSRFCookie(w, s.cookieSecure(r), 0, time.Time{})
		}

		if !validCSRFRequest(r) {
			writeJSON(w, http.StatusForbidden, apiResponse{OK: false, Error: "CSRF token missing or invalid"})
			return
		}

		next(w, r)
	}
}
