package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/report"
	"github.com/zerodha/fastglue"
)

// handleOverviewCounts retrieves general dashboard counts for all users.
func handleOverviewCounts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	counts, err := app.report.GetOverViewCounts()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(counts)
}

// handleOverviewCharts retrieves general dashboard chart data.
func handleOverviewCharts(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	charts, err := app.report.GetOverviewChart(days, app.setting.GetAppTimezone())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(charts)
}

// handleOverviewSLA retrieves SLA data for the dashboard.
func handleOverviewSLA(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	sla, err := app.report.GetOverviewSLA(days, app.setting.GetAppTimezone())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(sla)
}

// handleOverviewCSAT retrieves CSAT metrics for the dashboard.
func handleOverviewCSAT(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	csat, err := app.report.GetOverviewCSAT(days, app.setting.GetAppTimezone())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(csat)
}

// handleOverviewMessageVolume retrieves message volume metrics for the dashboard.
func handleOverviewMessageVolume(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	volume, err := app.report.GetOverviewMessageVolume(days, app.setting.GetAppTimezone())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(volume)
}

// handleOverviewTagDistribution retrieves tag distribution metrics for the dashboard.
func handleOverviewTagDistribution(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	tags, err := app.report.GetOverviewTagDistribution(days, app.setting.GetAppTimezone())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tags)
}

// handleIssuesReport returns bundled Pylon-style issue analytics.
func handleIssuesReport(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
		search  = string(r.RequestCtx.QueryArgs().Peek("account_search"))
		start   = string(r.RequestCtx.QueryArgs().Peek("start_date"))
		end     = string(r.RequestCtx.QueryArgs().Peek("end_date"))
	)
	payload, err := app.report.GetIssuesReport(report.IssuesRangeInput{
		Days:        days,
		StartDate:   start,
		EndDate:     end,
		AppTimezone: app.setting.GetAppTimezone(),
	}, search)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(payload)
}
