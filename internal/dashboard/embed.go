//go:build !dev

package dashboard

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embeddedFS embed.FS

// distFS wraps the embedded filesystem. Non-nil in production builds.
var distFS fs.FS = embeddedFS
