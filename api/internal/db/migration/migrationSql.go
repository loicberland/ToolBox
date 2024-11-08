package migration

import "embed"

//go:embed deploy/*.sql
var Deploy embed.FS

//go:embed revert/*.sql
var Revert embed.FS
