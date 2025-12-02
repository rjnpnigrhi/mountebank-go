package web

import (
	"embed"
	"io/fs"
)

//go:embed all:views all:public
var assets embed.FS

// GetAssets returns the embedded assets file system
func GetAssets() fs.FS {
	return assets
}
