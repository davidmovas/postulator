package wpplugin

import "embed"

const Root = "postulator-companion"

//go:embed all:postulator-companion
var Files embed.FS
