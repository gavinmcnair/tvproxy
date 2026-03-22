package defaults

import "embed"

//go:embed clients.json settings.json
var Assets embed.FS
