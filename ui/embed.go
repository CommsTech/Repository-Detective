package ui

import "embed"

//go:embed all:templates
var templateFS embed.FS

//go:embed all:static
var staticFS embed.FS

// StaticFS exposes embedded UI static assets for tests.
func StaticFS() embed.FS {
	return staticFS
}
