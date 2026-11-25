package delve

import (
	"testing"
	"time"
)

func TestDayIndexCountsUTCDays(t *testing.T) {
	epoch := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	if got := DayIndex(time.Date(2026, 11, 2, 23, 59, 0, 0, time.UTC), epoch); got != 0 {
		t.Fatalf("day 0 = %d", got)
	}
	if got := DayIndex(time.Date(2026, 11, 3, 0, 0, 1, 0, time.UTC), epoch); got != 1 {
		t.Fatalf("day 1 = %d", got)
	}
	// Before the epoch clamps at 0 (weekend visitors see day 0).
	if got := DayIndex(time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC), epoch); got != 0 {
		t.Fatalf("pre-epoch day = %d", got)
	}
}

func TestDailySeedDependsOnSecretAndDay(t *testing.T) {
	a := DailySeed([]byte("secret-one"), 4)
	b := DailySeed([]byte("secret-two"), 4)
	c := DailySeed([]byte("secret-one"), 5)
	if len(a) != 32 {
		t.Fatalf("seed length = %d", len(a))
	}
	if equal(a, b) || equal(a, c) {
		t.Fatal("seed must differ on secret or day")
	}
	if !equal(DailySeed([]byte("secret-one"), 4), a) {
		t.Fatal("seed must be deterministic")
	}
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
