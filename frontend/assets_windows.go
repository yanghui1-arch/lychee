//go:build windows && amd64

// Package frontend embeds shared frontend resources for the desktop backend.
package frontend

import "embed"

// SubtitleFonts includes the font shared with the preview and its license.
//
//go:embed src/assets/fonts/Roboto-Medium.ttf licenses/roboto-OFL.txt
var SubtitleFonts embed.FS
