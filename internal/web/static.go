package web

import "embed"

//go:embed all:dist
var frontend embed.FS
