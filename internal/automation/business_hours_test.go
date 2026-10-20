package automation

import (
	"errors"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
)

func TestOutsideBusinessHoursCondition(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skipf("no time zone data: %v", err)
	}
	hours := &bhmodels.BusinessHours{
		ID:       1,
		Hours:    []byte(`{"Monday":{"open":"09:00","close":"17:00"},"Friday":{"open":"09:00","close":"17:00"}}`),
		Holidays: []byte(`[{"name":"Day after Thanksgiving","date":"2026-11-27"}]`),
	}
	rule := func(value string) models.RuleDetail {
		return models.RuleDetail{Field: models.ConversationOutsideBusinessHours, FieldType: models.FieldTypeConversationField, Operator: models.RuleOperatorEquals, Value: value}
	}
	at := func(s string) func() time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, chicago)
		if err != nil {
			t.Fatal(err)
		}
		return func() time.Time { return v }
	}
	conv := createTestConversation()

	for _, tc := range []struct {
		name    string
		now     func() time.Time
		hours   *bhmodels.BusinessHours
		err     error
		outside bool
	}{
		{"inside hours", at("2026-10-19 10:00"), hours, nil, false},
		{"outside hours", at("2026-10-19 18:30"), hours, nil, true},
		{"weekend", at("2026-10-24 10:00"), hours, nil, true},
		{"holiday", at("2026-11-27 10:00"), hours, nil, true},
		{"no business hours configured", at("2026-10-24 10:00"), nil, nil, false},
		{"business hours unreadable", at("2026-10-24 10:00"), nil, errors.New("db down"), false},
	} {
		e := createTestEngine(&mockConversationStore{})
		e.now = tc.now
		e.SetDefaultBusinessHours(func() (*bhmodels.BusinessHours, *time.Location, error) {
			return tc.hours, chicago, tc.err
		})
		if got := e.evaluateRule(rule("true"), conv, nil); got != tc.outside {
			t.Errorf("%s: outside_business_hours = true is %v, want %v", tc.name, got, tc.outside)
		}
		if got := e.evaluateRule(rule("false"), conv, nil); got == tc.outside {
			t.Errorf("%s: outside_business_hours = false is %v, want %v", tc.name, got, !tc.outside)
		}
	}

	// An engine without a business hours reader treats every time as inside hours.
	if createTestEngine(&mockConversationStore{}).evaluateRule(rule("true"), conv, nil) {
		t.Error("outside_business_hours was true with no business hours reader")
	}
}
