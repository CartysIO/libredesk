package pylonimport

import (
	"testing"
	"time"
)

func TestPlanStatusSync(t *testing.T) {
	cut := time.Date(2026, 3, 15, 18, 0, 0, 0, time.UTC)
	imported := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)

	t.Run("updated when statuses differ", func(t *testing.T) {
		p := PlanStatusSync(imported, cut, true, false, StatusOpen, StatusResolved)
		if p.Action != StatusActionUpdated || !p.ShouldApply {
			t.Fatalf("got %+v", p)
		}
	})

	t.Run("unchanged when equal", func(t *testing.T) {
		p := PlanStatusSync(imported, cut, true, false, StatusOpen, StatusOpen)
		if p.Action != StatusActionUnchanged || !p.ShouldApply {
			t.Fatalf("got %+v", p)
		}
	})

	t.Run("skip native activity after cutover", func(t *testing.T) {
		p := PlanStatusSync(imported, cut, true, true, StatusOpen, StatusResolved)
		if p.Action != StatusActionSkipped || p.SkippedReason != "libredesk_active_after_cutover" {
			t.Fatalf("got %+v", p)
		}
	})

	t.Run("skip when imported after cutover end", func(t *testing.T) {
		late := cut.Add(time.Hour)
		p := PlanStatusSync(late, cut, true, false, StatusOpen, StatusResolved)
		if p.Warning != "imported_after_cutover_end" || p.SkippedReason != "imported_after_cutover_end" {
			t.Fatalf("got %+v", p)
		}
	})
}
