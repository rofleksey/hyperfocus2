// Package web embeds the compiled Vue frontend (web/dist) into the binary.
package web

import "embed"

// Dist holds the built frontend assets. The dist directory is populated by
// `npm run build` in web/ (see the Dockerfile / Makefile).
//
//go:embed all:dist
var Dist embed.FS
