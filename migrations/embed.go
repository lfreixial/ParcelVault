// Package migrations contains the SQL shipped with the ParcelVault binary.
package migrations

import "embed"

//go:embed sqlite/*.sql postgres/*.sql
var Files embed.FS
