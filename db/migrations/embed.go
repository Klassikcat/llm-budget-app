package migrations

import "embed"

// Files contains the SQL migration assets for the SQLite adapter.
//
//go:embed *.sql
var Files embed.FS
