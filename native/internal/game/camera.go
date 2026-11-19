package game

import (
	"math"

	"github.com/openkakutou/stage"
)

// DefaultCameraTension is Ikemen GO's [Camera] tension default (camera.go:85).
const DefaultCameraTension = 50

// CameraTension is the stage's [Camera] tension in stage units. A stage
// without a [Camera] section reads 0 and gets Ikemen GO's default.
func CameraTension(st *stage.Stage) int {
	if st.BGdef.Tension <= 0 {
		return DefaultCameraTension
	}
	return st.BGdef.Tension
}

// ElementScale is the scale element e is drawn at (BG scalestart); an unset
// one (0) draws at normal size, as stage.Parse's default of 1 does.
// shortcut: scaledelta only matters when the camera zooms, which this game
// does not do yet, and xscale only for parallax elements, which are not drawn
// as parallax yet (stage.go, commit 54d4148a).
func ElementScale(e stage.BGElement) (x, y float64) {
	x, y = e.ScaleStartX, e.ScaleStartY
	if x == 0 {
		x = 1
	}
	if y == 0 {
		y = 1
	}
	return x, y
}

// GroundLine is the view row of the fighters' ground: the stage's resolved
// zoffset scaled to the view under the stage's bottom alignment, or fallback
// for a stage that declares none.
func GroundLine(st *stage.Stage, scale, shift, fallback float64) float64 {
	if st.BGdef.ResolvedZOffset <= 0 {
		return fallback
	}
	return shift + scale*float64(st.BGdef.ResolvedZOffset)
}

// mugenLocalWidth is the localcoord width of a stage that declares none.
const mugenLocalWidth = 320

// StageScale is how many view pixels one stage unit spans: the view width
// over the stage's localcoord width (Ikemen GO stage.go:1772).
func StageScale(st *stage.Stage, viewW float64) float64 {
	w := st.BGdef.LocalCoordWidth
	if w <= 0 {
		w = mugenLocalWidth
	}
	return viewW / float64(w)
}

// StageGroundShift is how far down the stage is drawn when its scaled
// localcoord is shorter than the view: Ikemen GO bottom-aligns it (camera.go:149,
// overdrawlow 0). shortcut: a taller stage is not cropped (cutlow), shift 0.
func StageGroundShift(st *stage.Stage, scale, viewH float64) float64 {
	h := float64(st.BGdef.LocalCoordHeight) * scale
	if h <= 0 || h >= viewH {
		return 0
	}
	return viewH - h
}

// LayerOrigin is the view top-left of element e's sprite whose axis is
// axisX,axisY, with the camera at camX view pixels. The axis sits on the
// element's start point, scrolled against the camera by delta, around the
// view's horizontal centre (Ikemen GO stage.go:665 and 810, anim.go:1095).
// The sprite is drawn at the element's scalestart, so its axis offset grows
// with it.
func LayerOrigin(e stage.BGElement, axisX, axisY int, camX, scale, viewW, shift float64) (x, y float64) {
	sclX, sclY := ElementScale(e)
	sx := float64(e.StartX) - camX/scale*e.DeltaX
	x = viewW/2 + scale*(sx-float64(axisX)*sclX)
	y = shift + scale*(float64(e.StartY)-float64(axisY)*sclY)
	return x, y
}

// TileRun is where the copies of a sprite placed at pos, step apart, start
// along one axis and how many are drawn: tile 0 draws one, tile 1 repeats
// across the whole view, more repeats that many times from pos.
func TileRun(pos, step, view float64, tile int) (first float64, n int) {
	if tile == 0 || step <= 0 {
		return pos, 1
	}
	if tile > 1 {
		return pos, tile
	}
	first = pos - math.Ceil(pos/step)*step
	n = int(math.Ceil((view - first) / step))
	return first, n
}

// StageCamera follows the fighters horizontally as Ikemen GO's camera does
// with zoom off (camera.go:340-590): it moves only once a fighter comes
// within tension of a view edge, eases there, and stays within the bounds.
type StageCamera struct {
	halfWidth, tension, minX, maxX, startX float64
}

// NewStageCamera is st's camera for a viewW-wide view, tension in stage units.
func NewStageCamera(st *stage.Stage, tension int, scale, viewW float64) StageCamera {
	zoomOut := st.BGdef.ZoomOut
	if zoomOut <= 0 {
		zoomOut = 1
	}
	half := viewW / 2
	// minLeft/maxRight (camera.go:193-194), then XBound (camera.go:314).
	minLeft := float64(st.CameraBounds.Left)*scale - half/zoomOut
	maxRight := float64(st.CameraBounds.Right)*scale + half/zoomOut
	return StageCamera{halfWidth: half, tension: math.Max(0, float64(tension)*scale), minX: minLeft + half, maxX: maxRight - half, startX: float64(st.BGdef.CameraStartX) * scale}
}

// Start is the camera position at round start: the stage's [Camera] startx.
func (c StageCamera) Start() float64 { return c.startX }

// Follow is the camera's next position from x, given the leftmost and
// rightmost fighter positions; snap jumps straight to the target, as on a
// round's first frame.
func (c StageCamera) Follow(x, leftest, rightest float64, snap bool) float64 {
	minLeft, maxRight := c.minX-c.halfWidth, c.maxX+c.halfWidth
	oldLeft, oldRight := x-c.halfWidth, x+c.halfWidth
	targetLeft, targetRight := oldLeft, oldRight
	if leftest < targetLeft+c.tension {
		diff := targetLeft - math.Max(leftest-c.tension, minLeft)
		targetLeft = math.Max(leftest-c.tension, minLeft)
		targetRight = math.Max(oldRight-diff, math.Min(rightest+c.tension, maxRight))
	} else if rightest > targetRight-c.tension {
		diff := targetRight - math.Min(rightest+c.tension, maxRight)
		targetRight = math.Min(rightest+c.tension, maxRight)
		targetLeft = math.Min(oldLeft-diff, math.Max(leftest-c.tension, minLeft))
	}
	newLeft, newRight := targetLeft, targetRight
	if !snap {
		newLeft = ease(oldLeft, targetLeft, c.halfWidth*2)
		newRight = ease(oldRight, targetRight, c.halfWidth*2)
	}
	return math.Min(math.Max((newLeft+newRight)/2, c.minX), c.maxX)
}

// ease is Ikemen GO's three-pass edge smoothing at 60 fps, tensionvel 1
// (camera.go:552-583). shortcut: the fighters' velocity floor is not applied.
func ease(from, to, viewW float64) float64 {
	step := viewW / 3200
	v := from
	for range 3 {
		v += (to - v) * 0.05
		d := to - v
		if math.Abs(d) <= step {
			v = to
		} else if d > 0 {
			v += step
		} else {
			v -= step
		}
	}
	return v
}
