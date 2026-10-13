// Package db embeds the SQL migrations into the binary, so `netopology migrate` needs no files.
package db

import "embed"

// Migrations holds db/migrations/*.sql (goose format). Never edit a migration that has been
// merged; add a new numbered file instead.
//
//go:embed migrations/*.sql
var Migrations embed.FS
