package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC10_CLYR4PylonWorkforceFunctions(t *testing.T) {
	db := testutil.NewDB(t, "mig_clyr_4_pylon_fn")
	V2_9_0_RC10_CLYR4(db, nil, nil)

	var anchor string
	if err := db.Get(&anchor, `SELECT to_char(
		pylon_workforce_anchor_at('{"pylon_created_at": "2026-11-09T12:00:00Z"}'::jsonb, '2026-11-18T12:00:00Z'::timestamptz),
		'YYYY-MM-DD'
	)`); err != nil {
		t.Fatal(err)
	}
	if anchor != "2026-11-09" {
		t.Fatalf("anchor = %s, want 2026-11-09", anchor)
	}

	var fr *float64
	err := db.Get(&fr, `SELECT workforce_first_response_duration_sec(
		'2026-11-09T12:00:00Z'::timestamptz,
		'2026-11-09T16:00:00Z'::timestamptz,
		NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}
	if fr != nil {
		t.Fatalf("expected NULL FR for 4h auto-ack, got %v", *fr)
	}

	var res float64
	if err := db.Get(&res, `SELECT workforce_resolution_duration_sec(
		'{"pylon_resolution_seconds": 82380}'::jsonb,
		'2026-11-09T12:00:00Z'::timestamptz,
		NULL, NULL, NULL
	)`); err != nil {
		t.Fatal(err)
	}
	if res != 82380 {
		t.Fatalf("resolution = %v, want 82380", res)
	}
}
