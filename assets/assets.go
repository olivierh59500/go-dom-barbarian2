// Package assets embeds native artwork, animation tables and the YM soundtrack.
package assets

import "embed"

//go:embed original/*
var Files embed.FS
