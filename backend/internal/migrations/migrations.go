package migrations

import "embed"

// FS embeds the migration SQL files so tooling can apply them without a
// runtime path dependency.
//
//go:embed *.sql
var FS embed.FS
