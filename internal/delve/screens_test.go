package delve

import (
	"testing"
)

func TestGenerateFloorIsDeterministic(t *testing.T) {
	seed := DailySeed([]byte("test-secret"), 12)
	f1 := GenerateFloor(RandFromSeed(seed))
	f2 := GenerateFloor(RandFromSeed(seed))
	if f1.Theme != f2.Theme || f1.Width != f2.Width || f1.Height != f2.Height {
		t.Fatalf("floors diverge: %+v vs %+v", f1, f2)
	}
	if len(f1.KindAt) != len(f2.KindAt) {
		t.Fatalf("screen maps diverge: %d vs %d", len(f1.KindAt), len(f2.KindAt))
	}
	for coord, kind := range f1.KindAt {
		if f2.KindAt[coord] != kind {
			t.Fatalf("screen %v diverges: %s vs %s", coord, kind, f2.KindAt[coord])
		}
	}
}

func TestGenerateFloorInvariants(t *testing.T) {
	start := [2]int{2, 2}
	descents, safe := 0, 0
	for i := int64(0); i < 200; i++ {
		seed := DailySeed([]byte("test-secret"), i)
		f := GenerateFloor(RandFromSeed(seed))
		if f.ScreenAt(start[0], start[1]) != KindStart {
			t.Fatalf("day %d: start not at center", i)
		}
		for coord, kind := range f.KindAt {
			if coord[0] < 0 || coord[1] < 0 || coord[0] >= f.Width || coord[1] >= f.Height {
				t.Fatalf("day %d: screen %v out of bounds", i, coord)
			}
			switch kind {
			case KindDescent:
				descents++
			case KindSafe:
				safe++
			}
		}
		if descents != 2 || safe != 1 {
			t.Fatalf("day %d: want 2 descents + 1 safe, got %d + %d", i, descents, safe)
		}
		descents, safe = 0, 0
	}
}

func TestScreenAtBoundsAreOpen(t *testing.T) {
	f := Floor{Theme: "t", Width: 5, Height: 5, KindAt: map[[2]int]ScreenKind{[2]int{2, 2}: KindStart}}
	if f.ScreenAt(-1, 0) != KindOpen || f.ScreenAt(5, 5) != KindOpen {
		t.Fatal("out-of-bounds must read open")
	}
	if f.ScreenAt(0, 0) != KindOpen {
		t.Fatal("unmarked in-bounds screen must read open")
	}
}
