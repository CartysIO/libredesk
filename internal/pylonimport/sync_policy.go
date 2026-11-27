package pylonimport

import "time"

// StatusSyncAction is what a sync run does to conversation status.
type StatusSyncAction string

const (
	StatusActionUpdated   StatusSyncAction = "updated"
	StatusActionUnchanged StatusSyncAction = "unchanged"
	StatusActionSkipped   StatusSyncAction = "skipped"
)

// StatusSyncPlan describes a status change decision before or during sync.
type StatusSyncPlan struct {
	Action        StatusSyncAction
	SkippedReason string
	Warning       string
	PylonStatus   string
	CurrentStatus string
	ShouldApply   bool
}

// PlanStatusSync decides whether Pylon status should be copied onto a migrated conversation.
func PlanStatusSync(importedAt, cutoverEnd time.Time, importedAtOK bool, nativeAfterCutover bool, currentStatus, pylonStatus string) StatusSyncPlan {
	plan := StatusSyncPlan{
		Action:        StatusActionSkipped,
		PylonStatus:   pylonStatus,
		CurrentStatus: currentStatus,
	}
	if importedAtOK && importedAt.After(cutoverEnd) {
		plan.Warning = "imported_after_cutover_end"
		plan.SkippedReason = "imported_after_cutover_end"
		return plan
	}
	if nativeAfterCutover {
		plan.SkippedReason = "libredesk_active_after_cutover"
		return plan
	}
	plan.ShouldApply = true
	if currentStatus == pylonStatus {
		plan.Action = StatusActionUnchanged
		return plan
	}
	plan.Action = StatusActionUpdated
	return plan
}
