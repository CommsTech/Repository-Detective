package ui

import "embed"

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// StaticFS exposes embedded UI static assets for tests.
func StaticFS() embed.FS {
	return staticFS
}
