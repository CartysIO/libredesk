package pylonimport

import (
	"encoding/json"
	"strings"
	"time"
)

// Pylon issue state keys in time_in_status_seconds (order we replay for imported status activities).
var pylonStatusSegmentOrder = []string{
	"new",
	"open",
	"waiting_on_you",
	"in_progress",
	"waiting_on_customer",
	"on_hold",
	"closed",
}

// MapPylonStatusKey maps a Pylon time_in_status_seconds key to a Libredesk status name.
func MapPylonStatusKey(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "closed":
		return StatusResolved
	case "waiting_on_customer":
		return StatusOnCustomer
	case "on_hold":
		return StatusSnoozed
	case "waiting_on_you", "in_progress":
		return StatusOnYou
	default:
		return StatusOpen
	}
}

// conversationMeta builds conversations.meta for a new or merged Pylon issue.
func conversationMeta(is Issue) map[string]any {
	m := map[string]any{
		"pylon_number":     is.Number,
		"pylon_link":       is.Link,
		"pylon_source":     is.Source,
		"pylon_touches":    is.NumberOfTouches,
		"pylon_created_at": is.CreatedAt.UTC().Format(time.RFC3339),
	}
	if len(is.TimeInStatusSeconds) > 0 {
		m["pylon_time_in_status_seconds"] = is.TimeInStatusSeconds
	}
	if is.FirstResponseTime != nil {
		m["pylon_first_response_at"] = is.FirstResponseTime.UTC().Format(time.RFC3339)
	}
	if is.ResolutionTime != nil {
		m["pylon_resolution_at"] = is.ResolutionTime.UTC().Format(time.RFC3339)
	}
	if is.FirstResponseSeconds != nil {
		m["pylon_first_response_seconds"] = *is.FirstResponseSeconds
	}
	if is.ResolutionSeconds != nil {
		m["pylon_resolution_seconds"] = *is.ResolutionSeconds
	}
	if blob, err := json.Marshal(is); err == nil {
		var mirror map[string]any
		if json.Unmarshal(blob, &mirror) == nil {
			m["pylon"] = mirror
		}
	}
	return m
}
