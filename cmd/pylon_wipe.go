package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/abhinavxd/libredesk/internal/pylonimport"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
)

// runPylonWipe deletes every conversation and related Libredesk data (--pylon-wipe) so --pylon-import can run again.
// It touches only the database and configured upload store. Pass --confirm to execute; without it, only counts print.
func runPylonWipe(ctx context.Context, db *sqlx.DB, fs stuffbin.FileSystem, settings *setting.Manager) {
	opt := pylonimport.WipeOptions{Confirm: ko.Bool("confirm")}
	started := time.Now()

	w := &pylonimport.Wiper{DB: db, Opt: opt, Out: os.Stdout}
	if opt.Confirm {
		w.Media = initMedia(db, initI18n(fs), settings)
	}

	rep, err := w.Run(ctx)
	if rep != nil {
		rep.Print(os.Stdout)
	}
	if err != nil {
		log.Fatalf("pylon wipe: %v", err)
	}
	fmt.Printf("Done in %s.\n", time.Since(started).Round(time.Second))
}
