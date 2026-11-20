package ui

import (
	"github.com/openkakutou/screenpack"
)

// View is the size of the screen being drawn, in pixels.
type View struct{ W, H int }

// unit is the pixels in one screenpack unit, 1/100 of the screen height.
func (v View) unit() float32 { return float32(v.H) / 100 }

// referenceLines is the screen height an image's own pixels are sized for
// when an element gives no size: 1 image pixel = 1/240 of the height.
const referenceLines = 240

func (v View) rect() rect { return rect{0, 0, float32(v.W), float32(v.H)} }

type rect struct{ x, y, w, h float32 }

// Layout turns screens of a Pack into draw lists. It keeps no per-frame
// state, only a scratch slice reused between frames.
// shortcut: a screen's own `transition` (how it enters and leaves) is not
// applied by any screen; only element transitions are. Upgrade with the
// screen-change work, which knows when a screen is entered and left.
type Layout struct {
	pack *Pack
	rows []Row
}

// NewLayout returns a layout over pack.
func NewLayout(pack *Pack) *Layout { return &Layout{pack: pack} }

// labels turns plain entry labels into menu rows, in the layout's scratch.
func (l *Layout) labels(entries []string) []Row {
	l.rows = l.rows[:0]
	for _, e := range entries {
		l.rows = append(l.rows, Row{Label: e})
	}
	return l.rows
}

// anchorFactors are the fractions of a width and a height an anchor or pivot
// name stands for; an unknown or empty name is the center.
func anchorFactors(name string) (fx, fy float32) {
	switch name {
	case "topLeft":
		return 0, 0
	case "top":
		return 0.5, 0
	case "topRight":
		return 1, 0
	case "left":
		return 0, 0.5
	case "right":
		return 1, 0.5
	case "bottomLeft":
		return 0, 1
	case "bottom":
		return 0.5, 1
	case "bottomRight":
		return 1, 1
	}
	return 0.5, 0.5
}

// place puts a w x h box inside parent: its pivot (the anchor's name when
// unset) on the anchor point, moved by the offset. Every screenpack layout
// goes through here.
func place(parent rect, lay *screenpack.Layout, w, h, unit float32) rect {
	pivot := lay.Pivot
	if pivot == "" {
		pivot = lay.Anchor
	}
	ax, ay := anchorFactors(lay.Anchor)
	px, py := anchorFactors(pivot)
	return rect{
		x: parent.x + parent.w*ax + float32(lay.Offset[0])*unit - w*px,
		y: parent.y + parent.h*ay + float32(lay.Offset[1])*unit - h*py,
		w: w,
		h: h,
	}
}

// boxSize is the size in pixels of an image of natural w x h pixels with a
// layout size in units: a side set to null keeps the aspect ratio, and no
// size at all is the natural size at 1/240 of the height per image pixel.
func boxSize(size *[2]*float64, naturalW, naturalH float32, view View) (w, h float32) {
	unit := view.unit()
	scale := float32(view.H) / referenceLines
	if size == nil {
		return naturalW * scale, naturalH * scale
	}
	switch {
	case size[0] != nil && size[1] != nil:
		return float32(*size[0]) * unit, float32(*size[1]) * unit
	case size[0] != nil:
		w = float32(*size[0]) * unit
		return w, w * naturalH / naturalW
	case size[1] != nil:
		h = float32(*size[1]) * unit
		return h * naturalW / naturalH, h
	}
	return naturalW * scale, naturalH * scale
}

// parseColor reads #rrggbb or #rrggbbaa, its alpha multiplied by alpha. A
// malformed color (the screenpack loader refuses them) is transparent.
func parseColor(s string, alpha float32) Color {
	if (len(s) != 7 && len(s) != 9) || s[0] != '#' {
		return Color{}
	}
	var channel [4]float32
	channel[3] = 1
	for i := 0; i < (len(s)-1)/2; i++ {
		hi, lo := hexDigit(s[1+2*i]), hexDigit(s[2+2*i])
		channel[i] = float32(hi*16+lo) / 255
	}
	return Color{R: channel[0], G: channel[1], B: channel[2], A: channel[3] * alpha}
}

func hexDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}
