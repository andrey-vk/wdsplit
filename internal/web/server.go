// Package web is wdsplit's HTTP server: admin authentication today, the
// rest of the admin/end-user API as it's built.
package web

import (
	"net"
	"net/http"
	"time"

	"github.com/andrey-vk/wdsplit/internal/config"
)

const adminCookieName = "wdsplit_admin"

type Server struct {
	settings     *config.Settings
	loginLimiter *rateLimiter
	mux          *http.ServeMux
}

func NewServer(settings *config.Settings) *Server {
	s := &Server{
		settings:     settings,
		loginLimiter: newRateLimiter(5, time.Minute),
		mux:          http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/admin/login", s.handleLogin)
	s.mux.HandleFunc("GET /api/admin/me", s.requireAdmin(s.handleMe))
	s.mux.HandleFunc("POST /api/admin/logout", s.handleLogout)
}

func (s *Server) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// cookieSecure decides the Secure flag for admin cookies: an explicit
// setting wins, otherwise it follows whether the request itself arrived
// over TLS.
func (s *Server) cookieSecure(r *http.Request) bool {
	switch s.settings.AdminCookieSecure.Get() {
	case "true":
		return true
	case "false":
		return false
	}
	return r.TLS != nil
}
