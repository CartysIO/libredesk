package businesshours

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/business_hours/models"
)

func TestIsOpen(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skipf("no time zone data: %v", err)
	}
	bh := models.BusinessHours{
		Hours:    []byte(`{"Monday":{"open":"09:00","close":"17:00"},"Tuesday":{"open":"09:00","close":"17:00"},"Friday":{"open":"09:00","close":"17:00"}}`),
		Holidays: []byte(`[{"name":"Thanksgiving","date":"2026-11-26"},{"name":"Day after","date":"2026-11-27"}]`),
	}
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, chicago)
		if err != nil {
			t.Fatal(err)
		}
		return v.UTC()
	}
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"inside", at("2026-10-19 10:30"), true},
		{"at open", at("2026-10-19 09:00"), true},
		{"before open", at("2026-10-19 08:59"), false},
		{"at close", at("2026-10-19 17:00"), false},
		{"evening, already the next day in UTC", at("2026-10-19 20:00"), false},
		{"day without hours", at("2026-10-21 10:00"), false},
		{"weekend", at("2026-10-24 10:00"), false},
		{"holiday", at("2026-11-27 10:00"), false},
	}
	for _, tc := range cases {
		got, err := IsOpen(bh, tc.t, chicago)
		if err != nil || got != tc.want {
			t.Errorf("%s: got %v, %v; want %v", tc.name, got, err, tc.want)
		}
	}

	if open, _ := IsOpen(models.BusinessHours{IsAlwaysOpen: true}, at("2026-10-24 03:00"), chicago); !open {
		t.Error("always open business hours were closed")
	}
	if _, err := IsOpen(models.BusinessHours{Hours: []byte(`{"Monday":{"open":"9am","close":"17:00"}}`)}, at("2026-10-19 10:00"), chicago); err == nil {
		t.Error("invalid open time accepted")
	}
}
