package openwrtdiskioapi

import (
	"embed"
)

// please remember edit
// FrontendDistPath
// when go:embed path changed

const FrontendDistPath = "frontend/dist"

//go:embed frontend/dist
var WebEmb embed.FS
