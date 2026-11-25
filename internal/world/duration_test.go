package world

import "testing"

func TestAdvanceCompletesInOrder(t *testing.T) {
	queue := []Action{
		{Kind: "scout", Minutes: 45},
		{Kind: "chop", Minutes: 15},
	}
	completed, rest := Advance(queue, 60)
	if len(completed) != 2 || completed[0].Kind != "scout" || completed[1].Kind != "chop" {
		t.Fatalf("want scout then chop complete, got %+v", completed)
	}
	if len(rest) != 0 {
		t.Fatalf("queue should be empty, got %+v", rest)
	}
}

func TestAdvanceCarriesPartialProgress(t *testing.T) {
	queue := []Action{
		{Kind: "scout", Minutes: 45},
		{Kind: "fortify", Minutes: 120},
	}
	completed, rest := Advance(queue, 60)
	if len(completed) != 1 || completed[0].Kind != "scout" {
		t.Fatalf("scout should complete, got %+v", completed)
	}
	if len(rest) != 1 || rest[0].Kind != "fortify" || rest[0].Done != 15 {
		t.Fatalf("fortify should carry 15 minutes, got %+v", rest)
	}
	// The same tick math under a faster cadence must resume, not restart.
	second, _ := Advance(rest, 105)
	if len(second) != 1 || second[0].Kind != "fortify" || second[0].Done != 120 {
		t.Fatalf("fortify should finish with resumed progress, got %+v", second)
	}
}

func TestAdvanceZeroAndNegativeDelta(t *testing.T) {
	queue := []Action{{Kind: "scout", Minutes: 45}}
	completed, rest := Advance(queue, 0)
	if len(completed) != 0 || len(rest) != 1 || rest[0].Done != 0 {
		t.Fatalf("zero delta must not move the queue, got %+v / %+v", completed, rest)
	}
	_, rest2 := Advance(queue, -5)
	if len(rest2) != 1 || rest2[0].Done != 0 {
		t.Fatalf("negative delta must be inert, got %+v", rest2)
	}
}

func TestClampDelta(t *testing.T) {
	cases := []struct {
		delta int
		max   int
		want  int
	}{
		// The four cases: inside the cap, outage clamped to the cap,
		// max 0 disabling the cap, and clock-backwards being inert.
		{delta: 1440, max: 2880, want: 1440},
		{delta: 5000, max: 2880, want: 2880},
		{delta: 500, max: 0, want: 500},
		{delta: -10, max: 2880, want: 0},
	}
	for _, c := range cases {
		if got := ClampDelta(c.delta, c.max); got != c.want {
			t.Fatalf("ClampDelta(%d,%d) = %d, want %d", c.delta, c.max, got, c.want)
		}
	}
}
