package migrations

import "embed"

// FS contains service-backend database migrations.
//
//go:embed *.sql
var FS embed.FS
