package businesshours

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/business_hours/models"
)

// IsOpen reports whether t falls within the business hours, read in the time zone loc (Clyr fork).
// A holiday or a day without hours is closed. The open time is inclusive and the close time exclusive.
// ponytail: hours that close after midnight (close before open) count as closed; split them per day if needed.
func IsOpen(bh models.BusinessHours, t time.Time, loc *time.Location) (bool, error) {
	if bh.IsAlwaysOpen {
		return true, nil
	}
	local := t.In(loc)
	if len(bh.Holidays) > 0 {
		var holidays []models.Holiday
		if err := json.Unmarshal(bh.Holidays, &holidays); err != nil {
			return false, fmt.Errorf("parsing holidays: %w", err)
		}
		today := local.Format(time.DateOnly)
		for _, h := range holidays {
			if h.Date == today {
				return false, nil
			}
		}
	}
	var hours map[string]models.WorkingHours
	if err := json.Unmarshal(bh.Hours, &hours); err != nil {
		return false, fmt.Errorf("parsing working hours: %w", err)
	}
	day, ok := hours[local.Weekday().String()]
	if !ok {
		return false, nil
	}
	open, err := time.ParseInLocation("15:04", day.Open, loc)
	if err != nil {
		return false, fmt.Errorf("invalid open time %q: %w", day.Open, err)
	}
	shut, err := time.ParseInLocation("15:04", day.Close, loc)
	if err != nil {
		return false, fmt.Errorf("invalid close time %q: %w", day.Close, err)
	}
	minute := local.Hour()*60 + local.Minute()
	return minute >= open.Hour()*60+open.Minute() && minute < shut.Hour()*60+shut.Minute(), nil
}
