package scripts

import "embed"

//go:embed projections/*.nix
var Projections embed.FS
