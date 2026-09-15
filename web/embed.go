package web

import "embed"

// Assets is populated by npm run build before compiling the release executable.
//
//go:embed all:dist
var Assets embed.FS
