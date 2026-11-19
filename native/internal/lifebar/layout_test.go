package lifebar

import (
	"testing"

	"github.com/openkakutou/character/air"
)

func TestClipsAscendingTrailStartsAfterFront(t *testing.T) {
	b := Bar{RangeX: [2]int{0, 99}}
	front, trail := b.Clips(1, 320, 240, 0.5, 0.8)
	if front != (Rect{0, 0, 50, 240}) {
		t.Errorf("front = %+v", front)
	}
	if trail != (Rect{50, 0, 30, 240}) {
		t.Errorf("trail = %+v", trail)
	}
}

func TestClipsDescendingTrailKeepsItsLeftPart(t *testing.T) {
	b := Bar{RangeX: [2]int{99, 0}}
	front, trail := b.Clips(1, 320, 240, 0.5, 0.8)
	if front != (Rect{50, 0, 50, 240}) || trail != (Rect{20, 0, 30, 240}) {
		t.Errorf("front %+v trail %+v", front, trail)
	}
}

func TestClipsVerticalAndUnranged(t *testing.T) {
	b := Bar{RangeY: [2]int{0, 9}}
	front, trail := b.Clips(2, 320, 240, 0.5, 1)
	if front != (Rect{0, 0, 320, 10}) || trail != (Rect{0, 10, 320, 10}) {
		t.Errorf("vertical front %+v trail %+v", front, trail)
	}
	front, trail = Bar{}.Clips(1, 320, 240, 0.3, 0.6)
	if front != (Rect{0, 0, 320, 240}) || trail.W != 0 {
		t.Errorf("no range: front %+v trail %+v", front, trail)
	}
}

func TestPlaceUsesAxisOffsetAndFacing(t *testing.T) {
	e := Element{Offset: [2]float64{10, 4}, Facing: 1, Scale: [2]float64{1, 1}}
	fr := air.Frame{X: 2, Y: 1}
	r, flip := Place(e, fr, [2]int{3, 5}, [2]int{20, 8}, [2]int{100, 0}, 0.5)
	if flip || r != (Rect{54.5, 0, 10, 4}) {
		t.Errorf("right-facing = %+v %v", r, flip)
	}
	e.Facing = -1
	r, flip = Place(e, fr, [2]int{3, 5}, [2]int{20, 8}, [2]int{100, 0}, 0.5)
	if !flip || r != (Rect{45.5, 0, 10, 4}) {
		t.Errorf("mirrored = %+v %v", r, flip)
	}
	fr.Flip = air.FlipH
	if _, flip = Place(e, fr, [2]int{3, 5}, [2]int{20, 8}, [2]int{}, 1); flip {
		t.Error("an H frame on a mirrored element is unmirrored")
	}
}
