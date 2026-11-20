package ui

import (
	"unicode"

	"github.com/openkakutou/screenpack"
	"golang.org/x/image/font"
)

// styled is a text style ready to measure and draw at a screen height.
type styled struct {
	style screenpack.TextStyle
	font  FontRef
	px    int
	face  font.Face
	unit  float32
}

// styled looks a text style up in sc. ok is false when its font is unknown.
// The pixel size is the style's size in units, rounded: the renderer
// rasterizes glyphs at the same whole size, so measuring matches drawing.
func (l *Layout) styled(sc scope, id screenpack.StyleID, view View) (s styled, ok bool) {
	s.style = l.pack.style(sc, id)
	s.font = FontRef{sc.src, s.style.Font}
	s.unit = view.unit()
	s.px = max(1, int(float32(s.style.Size)*s.unit+0.5))
	s.face = l.pack.Face(s.font, s.px)
	return s, s.face != nil
}

// rune is the character drawn for r.
func (s styled) rune(r rune) rune {
	if s.style.Uppercase {
		return unicode.ToUpper(r)
	}
	return r
}

// width is the advance of text, in pixels.
// shortcut: no kerning; upgrade if a font's pairs show up in a screenpack.
func (s styled) width(text string) float32 {
	var w float32
	for _, r := range text {
		adv, _ := s.face.GlyphAdvance(s.rune(r))
		w += float32(adv) / 64
	}
	return w
}

// outlineSteps are the eight directions an outline is drawn in.
var outlineSteps = [8][2]float32{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

// emit adds a text run with its left edge at left and the line centered
// vertically on centerY: the shadow first, then the outline, then the fill.
func (s styled) emit(dst *DrawList, text string, left, centerY, alpha float32) {
	metrics := s.face.Metrics()
	baseline := centerY + float32(metrics.Ascent-metrics.Descent)/128
	first := len(dst.Glyphs)
	var pen float32
	for _, r := range text {
		r = s.rune(r)
		dst.Glyphs = append(dst.Glyphs, Glyph{R: r, X: pen})
		adv, _ := s.face.GlyphAdvance(r)
		pen += float32(adv) / 64
	}
	n := len(dst.Glyphs) - first
	run := Op{Kind: OpText, Font: s.font, Px: s.px, First: first, N: n}
	if sh := s.style.Shadow; sh != nil {
		run.X, run.Y = left+float32(sh.Offset[0])*s.unit, baseline+float32(sh.Offset[1])*s.unit
		run.Color = parseColor(sh.Color, alpha)
		dst.Ops = append(dst.Ops, run)
	}
	if ol := s.style.Outline; ol != nil {
		width := float32(ol.Width) * s.unit
		run.Color = parseColor(ol.Color, alpha)
		for _, step := range outlineSteps {
			run.X, run.Y = left+step[0]*width, baseline+step[1]*width
			dst.Ops = append(dst.Ops, run)
		}
	}
	run.X, run.Y, run.Color = left, baseline, parseColor(s.style.Color, alpha)
	dst.Ops = append(dst.Ops, run)
}

// text draws a text placed by a layout inside parent: its box is the text's
// own width by the style's line height, so the style's align changes nothing
// for it (align matters between the entries of a menu). A tint ("#rrggbb")
// replaces the style's color; "" keeps it.
func (l *Layout) text(dst *DrawList, view View, sc scope, lay *screenpack.Layout, style screenpack.StyleID, text string, parent rect, alpha float32, tint string) {
	s, ok := l.styled(sc, style, view)
	if !ok {
		return
	}
	if tint != "" {
		s.style.Color = tint
	}
	box := place(parent, lay, s.width(text), float32(s.style.Size)*s.unit, s.unit)
	s.emit(dst, text, box.x, box.y+box.h/2, alpha)
}

// alignFactor is how far along a line of free width the text starts: the
// fraction of the free width left of a text aligned left, center or right.
func alignFactor(align string) float32 {
	switch align {
	case "center":
		return 0.5
	case "right":
		return 1
	}
	return 0
}

// warningColor tints what the game flags (a binding shared by two actions, a
// settings notice): the format has no warning style, so it is the one color
// the layout sets itself.
const warningColor = "#dc3c3c"
