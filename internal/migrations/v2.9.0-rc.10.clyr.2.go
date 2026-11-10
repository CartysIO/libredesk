package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC10_CLYR2 marks the release that adds messages:write_backdated (registered in application code; no schema change).
func V2_9_0_RC10_CLYR2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	return nil
}
