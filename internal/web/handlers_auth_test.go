package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrey-vk/wdsplit/internal/config"
)

type fakeStore struct {
	values map[string]string
}

func (f *fakeStore) GetAll(context.Context) (map[string]string, error) {
	return f.values, nil
}

func (f *fakeStore) Save(_ context.Context, key, value string) error {
	f.values[key] = value
	return nil
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("WDSPLIT_ADMIN_PASSWORD", "hunter2")
	t.Setenv("WDSPLIT_SESSION_SECRET", "test-session-secret")
	t.Setenv("WDSPLIT_ADMIN_COOKIE_SECURE", "false")

	cfg, err := config.Load(context.Background(), &fakeStore{values: map[string]string{}}, nil)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return NewServer(cfg, nil)
}

func doLogin(t *testing.T, s *Server, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"password": password})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

func TestLoginSuccess(t *testing.T) {
	s := newTestServer(t)
	w := doLogin(t, s, "hunter2")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var gotSession, gotCSRF bool
	for _, c := range w.Result().Cookies() {
		switch c.Name {
		case adminCookieName:
			gotSession = true
			if !c.HttpOnly {
				t.Error("admin session cookie is not HttpOnly")
			}
		case csrfCookieName:
			gotCSRF = true
			if c.HttpOnly {
				t.Error("CSRF cookie is HttpOnly, want readable by JS")
			}
		}
	}
	if !gotSession {
		t.Error("login response did not set the admin session cookie")
	}
	if !gotCSRF {
		t.Error("login response did not set the CSRF cookie")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	s := newTestServer(t)
	w := doLogin(t, s, "wrong-password")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if len(w.Result().Cookies()) != 0 {
		t.Error("a failed login set cookies, want none")
	}
}

func TestLoginRateLimited(t *testing.T) {
	s := newTestServer(t)
	for range 5 {
		doLogin(t, s, "wrong-password")
	}
	w := doLogin(t, s, "wrong-password")
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status after exceeding the login limit = %d, want 429", w.Code)
	}
}

func TestMeRequiresAuth(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestMeWithValidSession(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	for _, c := range loginResp.Result().Cookies() {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestLogoutClearsCookies(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range loginResp.Result().Cookies() {
		switch c.Name {
		case adminCookieName:
			sessionCookie = c
		case csrfCookieName:
			csrfCookie = c
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/logout", nil)
	req.AddCookie(sessionCookie)
	req.AddCookie(csrfCookie)
	req.Header.Set(csrfHeaderName, csrfCookie.Value)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	for _, c := range w.Result().Cookies() {
		if c.Name == adminCookieName && c.MaxAge >= 0 {
			t.Errorf("logout did not clear the admin cookie (MaxAge = %d)", c.MaxAge)
		}
	}
}

func TestRequireAdminRejectsMutatingRequestWithoutCSRFHeader(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	called := false
	protected := s.requireAdmin(func(http.ResponseWriter, *http.Request) { called = true })

	req := httptest.NewRequest(http.MethodPost, "/api/admin/whatever", nil)
	for _, c := range loginResp.Result().Cookies() {
		req.AddCookie(c)
	}
	// Deliberately no X-CSRF-Token header.
	w := httptest.NewRecorder()
	protected(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
	if called {
		t.Error("inner handler was called despite missing CSRF header")
	}
}

func TestRequireAdminAllowsMutatingRequestWithMatchingCSRFHeader(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	var csrfValue string
	for _, c := range loginResp.Result().Cookies() {
		if c.Name == csrfCookieName {
			csrfValue = c.Value
		}
	}

	called := false
	protected := s.requireAdmin(func(http.ResponseWriter, *http.Request) { called = true })

	req := httptest.NewRequest(http.MethodPost, "/api/admin/whatever", nil)
	for _, c := range loginResp.Result().Cookies() {
		req.AddCookie(c)
	}
	req.Header.Set(csrfHeaderName, csrfValue)
	w := httptest.NewRecorder()
	protected(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !called {
		t.Error("inner handler was not called despite a matching CSRF header")
	}
}
