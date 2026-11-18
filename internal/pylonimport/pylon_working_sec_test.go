package pylonimport

import (
	"database/sql"
	"testing"
)

func TestPylonWorkingSec(t *testing.T) {
	t.Parallel()
	neg := -1
	ok := 42
	cases := []struct {
		in   *int
		want sql.NullInt32
	}{
		{nil, sql.NullInt32{}},
		{&neg, sql.NullInt32{}},
		{&ok, sql.NullInt32{Int32: 42, Valid: true}},
	}
	for _, tc := range cases {
		got := pylonWorkingSec(tc.in)
		if got.Valid != tc.want.Valid || got.Int32 != tc.want.Int32 {
			t.Fatalf("pylonWorkingSec(%v) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}
