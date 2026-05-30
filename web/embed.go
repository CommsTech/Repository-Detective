package web

import "embed"

// Static assets for the onboarding UI.
//
//go:embed static/*
var Static embed.FS
