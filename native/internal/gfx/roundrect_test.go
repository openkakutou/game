package gfx

import (
	"math"
	"testing"
)

// area sums the triangles of xy pairs, 3 vertices each.
func area(xy []float32) float64 {
	a := 0.0
	for i := 0; i+5 < len(xy); i += 6 {
		x0, y0, x1, y1, x2, y2 := xy[i], xy[i+1], xy[i+2], xy[i+3], xy[i+4], xy[i+5]
		a += math.Abs(float64((x1-x0)*(y2-y0)-(x2-x0)*(y1-y0))) / 2
	}
	return a
}

func TestRoundedTriangles_CoverTheRoundedArea(t *testing.T) {
	// 100x60 with radius 10: the box minus 4 corners of (1-pi/4)*r^2.
	want := 100*60 - 4*(1-math.Pi/4)*100
	got := area(roundedTriangles(0, 0, 100, 60, 10, nil))
	if math.Abs(got-want) > want*0.005 {
		t.Fatalf("area = %v, want about %v", got, want)
	}
}

func TestRoundedTriangles_StayInsideTheBox(t *testing.T) {
	xy := roundedTriangles(5, 7, 40, 20, 10, nil)
	for i := 0; i < len(xy); i += 2 {
		if xy[i] < 5-1e-3 || xy[i] > 45+1e-3 || xy[i+1] < 7-1e-3 || xy[i+1] > 27+1e-3 {
			t.Fatalf("vertex %v,%v leaves the box", xy[i], xy[i+1])
		}
	}
}

func TestRoundedTriangles_ReuseDst(t *testing.T) {
	buf := roundedTriangles(0, 0, 10, 10, 2, nil)
	again := roundedTriangles(0, 0, 10, 10, 2, buf)
	if &again[0] != &buf[0] {
		t.Fatal("a buffer with room must be reused, not reallocated")
	}
}
