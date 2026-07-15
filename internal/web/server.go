// Package web is wdsplit's HTTP server: admin authentication today, the
// rest of the admin/end-user API as it's built.
package web

import (
	"io/fs"
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

// NewServer builds the HTTP server. staticFS is the embedded frontend
// build (see spaembed.go) and is served for anything that isn't an /api/
// route, with SPA fallback to index.html. Pass nil to skip registering a
// static handler entirely — used by tests and by local dev, where the
// Vite dev server serves the frontend directly and only proxies /api/
// calls to this server (see ~/wdsplit-dev.sh).
func NewServer(settings *config.Settings, staticFS fs.FS) *Server {
	s := &Server{
		settings:     settings,
		loginLimiter: newRateLimiter(5, time.Minute),
		mux:          http.NewServeMux(),
	}
	s.routes(staticFS)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes(staticFS fs.FS) {
	s.mux.HandleFunc("POST /api/admin/login", s.handleLogin)
	s.mux.HandleFunc("GET /api/admin/me", s.requireAdmin(s.handleMe))
	s.mux.HandleFunc("POST /api/admin/logout", s.handleLogout)

	if staticFS != nil {
		s.mux.Handle("/", spaHandler(staticFS))
	}
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
