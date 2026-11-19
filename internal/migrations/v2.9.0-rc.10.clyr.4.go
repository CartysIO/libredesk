package migrations

import (
	_ "embed"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

//go:embed pylon_workforce_functions.sql
var pylonWorkforceFunctionsSQL string

// V2_9_0_RC10_CLYR4 adds SQL helpers for Pylon Workforce SLA parity in report queries.
func V2_9_0_RC10_CLYR4(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(strings.TrimSpace(pylonWorkforceFunctionsSQL))
	return err
}
