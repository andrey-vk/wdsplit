package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return body
}

func authedRequest(t *testing.T, loginResp *httptest.ResponseRecorder, method, path string, body []byte) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	var csrf string
	for _, c := range loginResp.Result().Cookies() {
		req.AddCookie(c)
		if c.Name == csrfCookieName {
			csrf = c.Value
		}
	}
	req.Header.Set(csrfHeaderName, csrf)
	return req
}

func TestListSettingsRequiresAuth(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestListSettingsReturnsAllSettingsWithSecretsHidden(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	req := authedRequest(t, loginResp, http.MethodGet, "/api/admin/settings", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var dtos []settingDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(dtos) != 4 {
		t.Fatalf("got %d settings, want 4: %+v", len(dtos), dtos)
	}

	byKey := make(map[string]settingDTO)
	for _, d := range dtos {
		byKey[d.Key] = d
	}

	adminPassword, ok := byKey["admin_password"]
	if !ok {
		t.Fatal("admin_password not in response")
	}
	if adminPassword.Value != "" {
		t.Error("admin_password.Value should be empty (secret)")
	}
	if adminPassword.Editable {
		t.Error("admin_password.Editable should be false (env-only)")
	}
	if !adminPassword.Secret {
		t.Error("admin_password.Secret should be true")
	}

	sessionMaxAge, ok := byKey["session_max_age"]
	if !ok {
		t.Fatal("session_max_age not in response")
	}
	if sessionMaxAge.Value != "28800" {
		t.Errorf("session_max_age.Value = %q, want %q", sessionMaxAge.Value, "28800")
	}
	if !sessionMaxAge.Editable {
		t.Error("session_max_age.Editable should be true")
	}

	cookieSecure, ok := byKey["admin_cookie_secure"]
	if !ok {
		t.Fatal("admin_cookie_secure not in response")
	}
	if cookieSecure.Type != "select" || len(cookieSecure.Options) != 3 {
		t.Errorf("admin_cookie_secure = %+v, want type=select with 3 options", cookieSecure)
	}
}

func TestUpdateSettingsAppliesValidChange(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	body := mustMarshal(t, map[string]string{"session_max_age": "3600"})
	req := authedRequest(t, loginResp, http.MethodPut, "/api/admin/settings", body)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if got := s.settings.SessionMaxAge.Get(); got != 3600 {
		t.Errorf("SessionMaxAge = %d, want 3600", got)
	}
}

func TestUpdateSettingsRejectsUnknownKey(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	body := mustMarshal(t, map[string]string{"not_a_real_setting": "x"})
	req := authedRequest(t, loginResp, http.MethodPut, "/api/admin/settings", body)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestUpdateSettingsRejectsEnvOnlyKey(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	body := mustMarshal(t, map[string]string{"admin_password": "new-password"})
	req := authedRequest(t, loginResp, http.MethodPut, "/api/admin/settings", body)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestUpdateSettingsBatchIsAllOrNothing(t *testing.T) {
	s := newTestServer(t)
	loginResp := doLogin(t, s, "hunter2")

	// One valid change alongside one invalid value for the other editable
	// setting — the whole batch must be rejected, and the valid one must
	// not have been applied either.
	body := mustMarshal(t, map[string]string{
		"session_max_age":     "3600",
		"admin_cookie_secure": "not-a-valid-option",
	})
	req := authedRequest(t, loginResp, http.MethodPut, "/api/admin/settings", body)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if got := s.settings.SessionMaxAge.Get(); got != 28800 {
		t.Errorf("SessionMaxAge = %d, want unchanged default 28800 (batch must be all-or-nothing)", got)
	}
}

func TestUpdateSettingsRequiresAuth(t *testing.T) {
	s := newTestServer(t)
	body := mustMarshal(t, map[string]string{"session_max_age": "3600"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
