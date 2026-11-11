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

// runPylonImport copies the Pylon history into this database (--pylon-import). It reads Pylon with the token in
// PYLON_API_KEY, caches everything under --pylon-cache, and writes rows with SQL only, so imported data sends no
// email and fires no automation, SLA, webhook, Slack or notification event.
func runPylonImport(ctx context.Context, db *sqlx.DB, fs stuffbin.FileSystem, settings *setting.Manager) {
	opt := pylonimport.Options{
		DryRun:         ko.Bool("dry-run"),
		Limit:          ko.Int("limit"),
		InboxID:        ko.Int("pylon-inbox"),
		ReferenceFloor: ko.Int("pylon-reference-floor"),
		AgentAliases:   map[string]string{},
		MediaStore:     ko.MustString("upload.provider"),

		AllowMissingFiles:   ko.Bool("pylon-allow-missing-files"),
		Reconcile:           ko.Bool("pylon-reconcile"),
		VerifyIssuesInPylon: ko.Bool("pylon-verify-live"),
	}
	if s := ko.String("since"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			log.Fatalf("--since: want YYYY-MM-DD: %v", err)
		}
		opt.Since = t
	}
	for _, pair := range ko.Strings("pylon-agent-alias") {
		from, to, ok := strings.Cut(pair, "=")
		if !ok {
			log.Fatalf("--pylon-agent-alias: want pylon-email=libredesk-email, got %q", pair)
		}
		opt.AgentAliases[pylonimport.NormalizeEmail(from)] = pylonimport.NormalizeEmail(to)
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
	if !opt.Since.IsZero() {
		// Limit API issue listing to --since. Cached issues.json is filtered in-memory to the
		// same cutoff so message GETs stay bounded. Use a fresh --pylon-cache to refetch listing.
		fetcher.Oldest = opt.Since.UTC()
		log.Printf("pylon: fetching issues created on or after %s", fetcher.Oldest.Format("2006-01-02"))
	}
	started := time.Now()
	data, err := fetcher.Fetch(ctx)
	if err != nil {
		log.Fatalf("reading Pylon: %v", err)
	}
	log.Printf("pylon: %d users, %d accounts, %d contacts, %d issues (%s)", len(data.Users), len(data.Accounts), len(data.Contacts), len(data.Issues), time.Since(started).Round(time.Second))

	if opt.VerifyIssuesInPylon {
		if strings.TrimSpace(os.Getenv("PYLON_API_KEY")) == "" {
			log.Fatalf("pylon import: --pylon-verify-live needs PYLON_API_KEY to confirm each issue still exists in Pylon (or pass --pylon-verify-live=false to use the cache as-is)")
		}
		present, missing, err := fetcher.FilterPresentIssues(ctx, data.Issues)
		if err != nil {
			log.Fatalf("verifying issues in Pylon: %v", err)
		}
		for _, is := range missing {
			delete(data.Messages, is.ID)
			opt.SkippedNotInPylon = append(opt.SkippedNotInPylon,
				fmt.Sprintf("#%d %s (%s)", is.Number, clip(is.Title, 60), is.ID))
		}
		data.Issues = present
		log.Printf("pylon: importing %d issues after live verification (%d skipped as absent)", len(data.Issues), len(missing))
	}

	if _, err := db.Exec(`SET lock_timeout = '60s'; SET idle_in_transaction_session_timeout = '5min'; SET statement_timeout = '30min'`); err != nil {
		log.Fatalf("pylon import: database session: %v", err)
	}

	im := &pylonimport.Importer{DB: db, Opt: opt, Out: os.Stdout, Files: fetcher.ReadFile}
	if !opt.DryRun {
		im.Download = func(urls []string) map[string]error { return fetcher.DownloadFiles(ctx, urls) }
		im.Media = initMedia(db, initI18n(fs), settings)
	}

	rep, err := im.Run(ctx, data)
	if rep != nil {
		rep.Print(os.Stdout)
	}
	if err != nil {
		log.Fatalf("pylon import: %v", err)
	}
	fmt.Printf("Done in %s.\n", time.Since(started).Round(time.Second))
}
