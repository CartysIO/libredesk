package main

import (
	"encoding/json"
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
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

// handleIssuesReport returns bundled Pylon-style issue analytics (scalars and charts).
func handleIssuesReport(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
		start   = string(r.RequestCtx.QueryArgs().Peek("start_date"))
		end     = string(r.RequestCtx.QueryArgs().Peek("end_date"))
	)
	payload, err := app.report.GetIssuesReport(report.IssuesRangeInput{
		Days:        days,
		StartDate:   start,
		EndDate:     end,
		AppTimezone: app.setting.GetAppTimezone(),
	})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(payload)
}

// handleIssuesTopAccounts returns the sortable top-accounts table (single lightweight query).
func handleIssuesTopAccounts(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
		search  = string(r.RequestCtx.QueryArgs().Peek("account_search"))
		start   = string(r.RequestCtx.QueryArgs().Peek("start_date"))
		end     = string(r.RequestCtx.QueryArgs().Peek("end_date"))
		sortBy  = string(r.RequestCtx.QueryArgs().Peek("sort_by"))
		sortDir = string(r.RequestCtx.QueryArgs().Peek("sort_dir"))
	)
	payload, err := app.report.GetIssuesTopAccounts(
		report.IssuesRangeInput{
			Days:        days,
			StartDate:   start,
			EndDate:     end,
			AppTimezone: app.setting.GetAppTimezone(),
		},
		search,
		report.IssuesTopAccountsSort{By: sortBy, Dir: sortDir},
	)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(payload)
}

// handleSlaBreachesReport returns a paginated table of conversations with first-response or resolution SLA breaches.
func handleSlaBreachesReport(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
		search  = string(r.RequestCtx.QueryArgs().Peek("search"))
		start   = string(r.RequestCtx.QueryArgs().Peek("start_date"))
		end     = string(r.RequestCtx.QueryArgs().Peek("end_date"))
		sortBy  = string(r.RequestCtx.QueryArgs().Peek("sort_by"))
		sortDir = string(r.RequestCtx.QueryArgs().Peek("sort_dir"))
	)
	page, pageSize := slaBreachesPagination(r)
	payload, err := app.report.GetSlaBreaches(
		report.IssuesRangeInput{
			Days:        days,
			StartDate:   start,
			EndDate:     end,
			AppTimezone: app.setting.GetAppTimezone(),
		},
		search,
		report.SlaBreachesSort{By: sortBy, Dir: sortDir},
		page,
		pageSize,
	)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var rows []any
	if err := json.Unmarshal(payload.Rows, &rows); err != nil {
		return sendErrorEnvelope(r, err)
	}
	total := payload.Total
	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    rows,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: totalPages,
		Page:       page,
	})
}

func slaBreachesPagination(r *fastglue.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = report.SlaBreachesDefaultPageSize()
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// handleWorkforceReport returns bundled SLA and agent workload analytics.
func handleWorkforceReport(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
		start   = string(r.RequestCtx.QueryArgs().Peek("start_date"))
		end     = string(r.RequestCtx.QueryArgs().Peek("end_date"))
	)
	payload, err := app.report.GetWorkforceReport(report.IssuesRangeInput{
		Days:        days,
		StartDate:   start,
		EndDate:     end,
		AppTimezone: app.setting.GetAppTimezone(),
	})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(payload)
}
