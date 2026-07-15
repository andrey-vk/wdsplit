// Package spa embeds the built frontend (webgui/dist) into the binary for
// single-binary deployment, matching wdbgp's approach.
//
// The embed directive's path is relative to this file and can't traverse
// upward (no ../), so this file has to live at the module root, alongside
// webgui/ — it can't live under cmd/ or internal/.
//
// webgui/dist must exist and contain a real build before `go build`,
// `go vet`, or `go test` will succeed — run `cd webgui && npm install &&
// npm run build` first on a fresh checkout. CI (tests.yml) and the
// Dockerfile already build the frontend before the Go build step; the
// local dev script builds it once automatically if missing.
package spa

import (
	"embed"
	"io/fs"
)

//go:embed all:webgui/dist
var distFS embed.FS

// DistFS returns the embedded frontend build, rooted at webgui/dist
// itself (not webgui/dist/webgui/dist).
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "webgui/dist")
}
