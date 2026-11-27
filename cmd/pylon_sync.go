package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/pylonimport"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
)

// runPylonSync reconciles migrated conversations against live Pylon (--pylon-sync).
func runPylonSync(ctx context.Context, db *sqlx.DB, fs stuffbin.FileSystem, settings *setting.Manager) {
	cutoverStr := strings.TrimSpace(ko.String("pylon-cutover-end"))
	if cutoverStr == "" {
		log.Fatal("pylon sync: --pylon-cutover-end is required (RFC3339, e.g. 2026-03-15T18:00:00Z)")
	}
	cutoverEnd, err := time.Parse(time.RFC3339, cutoverStr)
	if err != nil {
		log.Fatalf("pylon sync: --pylon-cutover-end: want RFC3339: %v", err)
	}
	dryRun := ko.Bool("dry-run")
	reportPath := ko.String("pylon-sync-report")
	if reportPath == "" {
		reportPath = "pylon-sync-report.json"
	}

	imOpt := pylonimport.Options{
		DryRun:              dryRun,
		AllowMissingFiles:   ko.Bool("pylon-allow-missing-files") || dryRun,
		MediaStore:          ko.MustString("upload.provider"),
		AgentAliases:        map[string]string{},
		VerifyIssuesInPylon: ko.Bool("pylon-verify-live"),
	}
	syncOpt := pylonimport.SyncOptions{
		DryRun:              dryRun,
		Limit:               ko.Int("limit"),
		CutoverEnd:          cutoverEnd,
		AllowMissingFiles:   imOpt.AllowMissingFiles,
		VerifyIssuesInPylon: imOpt.VerifyIssuesInPylon,
		AgentAliases:        imOpt.AgentAliases,
		MediaStore:          imOpt.MediaStore,
		ReportPath:          reportPath,
	}
	if s := ko.String("since"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			log.Fatalf("--since: want YYYY-MM-DD: %v", err)
		}
		syncOpt.Since = t
	}
	for _, pair := range ko.Strings("pylon-agent-alias") {
		from, to, ok := strings.Cut(pair, "=")
		if !ok {
			log.Fatalf("--pylon-agent-alias: want pylon-email=libredesk-agent-email, got %q", pair)
		}
		imOpt.AgentAliases[pylonimport.NormalizeEmail(from)] = pylonimport.NormalizeEmail(to)
		syncOpt.AgentAliases = imOpt.AgentAliases
	}

	fetcher := &pylonimport.Fetcher{
		Dir:               ko.String("pylon-cache"),
		RefreshIssuesList: ko.Bool("pylon-refresh-issues"),
		Token: func() (string, error) {
			if t := strings.TrimSpace(os.Getenv("PYLON_API_KEY")); t != "" {
				return t, nil
			}
			return "", errors.New("the cache is incomplete and PYLON_API_KEY is not set")
		},
		Logf: func(format string, args ...any) { log.Printf(format, args...) },
	}
	if !syncOpt.Since.IsZero() {
		fetcher.Oldest = syncOpt.Since.UTC()
		log.Printf("pylon-sync: fetching issues created on or after %s", fetcher.Oldest.Format("2006-01-02"))
	}
	started := time.Now()
	data, err := fetcher.Fetch(ctx)
	if err != nil {
		log.Fatalf("reading Pylon: %v", err)
	}
	log.Printf("pylon-sync: %d issues in cache/API (%s)", len(data.Issues), time.Since(started).Round(time.Second))

	if syncOpt.VerifyIssuesInPylon {
		if strings.TrimSpace(os.Getenv("PYLON_API_KEY")) == "" {
			log.Fatal("pylon sync: --pylon-verify-live needs PYLON_API_KEY (or pass --pylon-verify-live=false)")
		}
		present, missing, err := pylonimport.FilterIssuesForSync(ctx, fetcher, data.Issues, true)
		if err != nil {
			log.Fatalf("verifying issues in Pylon: %v", err)
		}
		for _, is := range missing {
			delete(data.Messages, is.ID)
		}
		data.Issues = present
		log.Printf("pylon-sync: %d issues after live verification (%d absent)", len(present), len(missing))
	}

	if _, err := db.Exec(`SET lock_timeout = '60s'; SET idle_in_transaction_session_timeout = '5min'; SET statement_timeout = '30min'`); err != nil {
		log.Fatalf("pylon sync: database session: %v", err)
	}

	im := &pylonimport.Importer{DB: db, Opt: imOpt, Out: os.Stdout, Files: fetcher.ReadFile}
	if !dryRun {
		im.Download = func(urls []string) map[string]error { return fetcher.DownloadFiles(ctx, urls) }
		im.Media = initMedia(db, initI18n(fs), settings)
	}
	syncer := &pylonimport.Syncer{DB: db, Imp: im, Opt: syncOpt, Out: os.Stdout}

	rep, err := syncer.Run(ctx, data)
	if err != nil {
		log.Printf("pylon sync: %v", err)
	}
	log.Printf("pylon-sync: examined=%d status_updated=%d messages_inserted=%d errors=%d dry_run=%v",
		rep.Summary.Examined, rep.Summary.StatusUpdated, rep.Summary.MessagesInserted, rep.Summary.Errors, dryRun)
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("Done in %s.\n", time.Since(started).Round(time.Second))
}
