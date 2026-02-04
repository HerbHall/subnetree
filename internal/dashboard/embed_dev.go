//go:build dev

package dashboard

import "io/fs"

// distFS is nil in dev mode -- the handler returns 404 for all requests,
// allowing the Vite dev server proxy to serve assets instead.
var distFS fs.FS
