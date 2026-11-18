package migrations

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC10_CLYR4 adds SQL helpers for Pylon Workforce SLA parity in report queries.
func V2_9_0_RC10_CLYR4(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "pylon_workforce_functions.sql"))
	if err != nil {
		return err
	}
	_, err = db.Exec(strings.TrimSpace(string(b)))
	return err
}
