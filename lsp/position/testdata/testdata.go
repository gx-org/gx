// Package testdata stores the GX test files for position and LSP tests.
package testdata

import "embed"

// FS embeds the GX test files.
//
//go:embed *.gx
var FS embed.FS
