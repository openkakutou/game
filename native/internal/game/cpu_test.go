package game

import (
	"testing"

	"github.com/openkakutou/engine/v2/input"
)

// lowest and highest are draws that always return the low or the high end.
func lowest(lo, _ int) int  { return lo }
func highest(_, hi int) int { return hi }

func decideOnce(c *CPU) input.TickInput {
	var in input.TickInput
	c.Decide(&in)
	return in
}

func TestCPU_AWinningDrawJamsADirectionAndEveryButton(t *testing.T) {
	// Rand(1, chance) == 1 always: each timer is set (to 1), the direction
	// drawn is 0, up (input.go:1955-1997).
	in := decideOnce(NewCPU(4, lowest))
	if !in.Up || in.Down || in.Left || in.Right {
		t.Errorf("direction = %+v, want up alone", in)
	}
	for _, b := range []string{"a", "b", "c", "x", "y", "z", "s"} {
		if !in.Buttons[b] {
			t.Errorf("button %s not jammed", b)
		}
	}
}

func TestCPU_ALosingDrawPressesNothing(t *testing.T) {
	in := decideOnce(NewCPU(4, highest))
	if in.Up || in.Down || in.Left || in.Right {
		t.Errorf("direction = %+v, want none", in)
	}
	for b, held := range in.Buttons {
		if held {
			t.Errorf("button %s held", b)
		}
	}
}

func TestCPU_AJammedButtonIsHeldForItsDrawnTime(t *testing.T) {
	// The first draw sets every timer to 3 (Rand(1, 30) returns 3 here);
	// later draws lose: the buttons stay held 3 updates, then let go.
	first := true
	draw := func(lo, hi int) int {
		switch {
		case hi == 7: // direction
			return 2
		case first && (hi == 30 || hi == 60):
			return 3
		case first:
			return lo
		}
		return hi
	}
	c := NewCPU(4, draw)
	held := 0
	for range 6 {
		in := decideOnce(c)
		first = false
		if in.Buttons["a"] {
			held++
		}
	}
	if held != 3 {
		t.Fatalf("a held %d updates, want 3", held)
	}
}

func TestCPU_TheDirectionsFollowIkemenGOsEightWayDraw(t *testing.T) {
	// 0 U, 1 UR, 2 R, 3 DR, 4 D, 5 DL, 6 L, 7 UL (input.go:1999-2003).
	want := [8][4]bool{ // up, down, left, right
		{true, false, false, false}, {true, false, false, true}, {false, false, false, true}, {false, true, false, true},
		{false, true, false, false}, {false, true, true, false}, {false, false, true, false}, {true, false, true, false},
	}
	for dir := range 8 {
		draw := func(lo, hi int) int {
			if hi == 7 {
				return dir
			}
			return lo
		}
		in := decideOnce(NewCPU(4, draw))
		got := [4]bool{in.Up, in.Down, in.Left, in.Right}
		if got != want[dir] {
			t.Errorf("dir %d = %v, want %v", dir, got, want[dir])
		}
	}
}

func TestCPU_AHigherLevelJamsButtonsMoreOften(t *testing.T) {
	// chance = int((-11.25*level + 165) * 7): 1076 at level 1, 525 at level 8.
	chances := func(level float64) int {
		seen := 0
		c := NewCPU(level, func(lo, hi int) int {
			if hi != 7 && hi != 15 && hi != 30 && hi != 60 && hi != 3600 {
				seen = hi
			}
			return hi
		})
		decideOnce(c)
		return seen
	}
	if got := chances(1); got != 1076 {
		t.Errorf("level 1 chance = %d, want 1076", got)
	}
	if got := chances(8); got != 525 {
		t.Errorf("level 8 chance = %d, want 525", got)
	}
}

func TestCPU_ReusesTheButtonMapAndClearsStaleInput(t *testing.T) {
	c := NewCPU(4, highest)
	in := input.TickInput{Up: true, Buttons: map[string]bool{"a": true}}
	c.Decide(&in)
	if in.Up || in.Buttons["a"] {
		t.Errorf("stale input kept: %+v", in)
	}
	if allocs := testing.AllocsPerRun(50, func() { c.Decide(&in) }); allocs != 0 {
		t.Errorf("Decide allocates %v times per tick, want 0", allocs)
	}
}

func TestCPU_TheSameSeedPlaysTheSameInputs(t *testing.T) {
	a, b := NewCPU(6, SeededDraw(42)), NewCPU(6, SeededDraw(42))
	for tick := range 300 {
		if x, y := decideOnce(a), decideOnce(b); x.Up != y.Up || x.Left != y.Left || x.Buttons["a"] != y.Buttons["a"] || x.Buttons["z"] != y.Buttons["z"] {
			t.Fatalf("tick %d differs: %+v vs %+v", tick, x, y)
		}
	}
}
