package delve

import (
	"encoding/binary"
	"math/rand"
)

// ScreenKind drives spawn tables and renderers later; v1 keeps floors
// wide and simple (mostly open screens). Constrained generation and
// pathing arrive with themed dungeons, one at a time.
type ScreenKind string

const (
	KindOpen    ScreenKind = "open"
	KindStart   ScreenKind = "start"
	KindDescent ScreenKind = "descent" // hole down
	KindSafe    ScreenKind = "safe"    // stairway landing: organize items, no combat
)

// Floor is one dungeon floor: a grid of screens with a theme. Screens not
// present in KindAt are open.
type Floor struct {
	Theme  string
	Width  int
	Height int
	KindAt map[[2]int]ScreenKind
}

var themes = []string{"Sunken Crypt", "Cinder Warrens", "Mossy Rootway", "Flooded Gallery"}

// GenerateFloor deterministically builds a floor from a random source:
// two calls with the same seed produce identical floors. The layout seed
// never changes what combat rolls use; those are unseeded server RNG.
func GenerateFloor(rng *rand.Rand) Floor {
	const side = 5
	f := Floor{
		Theme:  themes[rng.Intn(len(themes))],
		Width:  side,
		Height: side,
		KindAt: make(map[[2]int]ScreenKind, 4),
	}
	start := [2]int{side / 2, side / 2}
	f.KindAt[start] = KindStart
	f.KindAt[pickDistinct(rng, f.KindAt, side)] = KindDescent
	f.KindAt[pickDistinct(rng, f.KindAt, side)] = KindDescent
	f.KindAt[pickDistinct(rng, f.KindAt, side)] = KindSafe
	return f
}

// RandFromSeed builds the layout RNG from a seed. Deterministic layout,
// never revealed beyond visited screens.
func RandFromSeed(seed []byte) *rand.Rand {
	var head [8]byte
	copy(head[:], seed)
	return rand.New(rand.NewSource(int64(binary.BigEndian.Uint64(head[:]))))
}

// ScreenAt reports the kind of a screen; out-of-bounds is open (the floor
// is imagined spacious at v1 edges; edges close in with themed dungeons).
func (f Floor) ScreenAt(x, y int) ScreenKind {
	if x < 0 || y < 0 || x >= f.Width || y >= f.Height {
		return KindOpen
	}
	if kind, ok := f.KindAt[[2]int{x, y}]; ok {
		return kind
	}
	return KindOpen
}

func pickDistinct(rng *rand.Rand, taken map[[2]int]ScreenKind, side int) [2]int {
	for {
		p := [2]int{rng.Intn(side), rng.Intn(side)}
		if _, ok := taken[p]; !ok {
			return p
		}
	}
}
