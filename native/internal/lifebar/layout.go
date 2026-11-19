package lifebar

import "github.com/openkakutou/character/air"

// Rect is an area of the game view, in view pixels.
type Rect struct{ X, Y, W, H float64 }

// Clips are the clip areas of a bar's front fill (at top) and trailing fill
// (at mid) in a view vw x vh, scale from the lifebar's localcoord (Ikemen GO
// LifeBar.draw, fightscreen.go:717-770, without scalefill). The trailing
// area only keeps the part the front does not cover.
func (b Bar) Clips(scale, vw, vh, top, mid float64) (front, trail Rect) {
	front = b.clip(scale, vw, vh, top)
	trail = b.clip(scale, vw, vh, mid)
	if b.RangeY != [2]int{} {
		if b.RangeY[0] < b.RangeY[1] {
			trail.Y += front.H
		}
		trail.H -= min(trail.H, front.H)
		return front, trail
	}
	if b.RangeX[0] < b.RangeX[1] {
		trail.X += front.W
	}
	trail.W -= min(trail.W, front.W)
	return front, trail
}

func (b Bar) clip(scale, vw, vh, fill float64) Rect {
	r := Rect{W: vw, H: vh}
	if b.RangeX != [2]int{} {
		x, w := FillSpan(b.Pos[0], b.RangeX, scale, fill)
		r.X, r.W = float64(x), float64(w)
	}
	if b.RangeY != [2]int{} {
		y, h := FillSpan(b.Pos[1], b.RangeY, scale, fill)
		r.Y, r.H = float64(y), float64(h)
	}
	return r
}

// Place is where a frame of e, owned by something at pos, lands in the view:
// its top-left corner and size, and whether it is mirrored. axis and size
// are the sprite's own (Ikemen GO Animation.Draw: the axis sits at
// pos + offset + the frame's offset, facing mirroring around it).
func Place(e Element, fr air.Frame, axis, size [2]int, pos [2]int, scale float64) (r Rect, flipX bool) {
	ax := float64(pos[0]) + e.Offset[0]
	ay := float64(pos[1]) + e.Offset[1]
	sx := e.Scale[0] * scale
	sy := e.Scale[1] * scale
	flipX = e.Facing < 0
	if fr.Flip == air.FlipH || fr.Flip == air.FlipHV {
		flipX = !flipX
	}
	fx := float64(fr.X)
	if e.Facing < 0 {
		fx = -fx
	}
	left := float64(axis[0])
	if flipX {
		left = float64(size[0] - axis[0])
	}
	r.X = ax*scale + fx*scale - left*sx
	r.Y = ay*scale + float64(fr.Y)*scale - float64(axis[1])*sy
	r.W = float64(size[0]) * sx
	r.H = float64(size[1]) * sy
	return r, flipX
}
