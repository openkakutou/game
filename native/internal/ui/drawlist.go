// Package ui turns a resolved screenpack screen and what the game shows into
// a flat draw list (rects, image quads, text runs). It draws nothing itself:
// the renderer executes the list. Units, anchors, pivots, offsets and sizes
// follow the screenpack format (docs/format.md), and are applied only here.
package ui

import "github.com/openkakutou/screenpack"

// Color is an RGBA color with 0..1 components.
type Color struct{ R, G, B, A float32 }

// Asset names a file of the pack the element came from (FromPack) or of the
// built-in default pack (FromDefault); Path is relative to that pack.
type Asset struct {
	Source screenpack.Source
	Path   string
}

// FontRef names a font of one of the two packs.
type FontRef struct {
	Source screenpack.Source
	ID     screenpack.FontID
}

// OpKind says which fields of an Op are used.
type OpKind uint8

const (
	// OpRect fills X, Y, W, H with Color.
	OpRect OpKind = iota
	// OpImage draws Asset's texture region U0,V0..U1,V1 (normalized, and
	// beyond 1 when the image repeats) into X, Y, W, H, tinted by Color.
	OpImage
	// OpText draws Glyphs[First:First+N] in Font at Px pixels, tinted by
	// Color; X, Y is the pen's origin on the baseline and each glyph's X is
	// its pen offset from it.
	OpText
)

// Glyph is one character of a text run and its pen offset.
type Glyph struct {
	R rune
	X float32
}

// Op is one thing to draw, in pixels of the view. A small tagged struct, so
// a frame allocates no value per op.
type Op struct {
	Kind           OpKind
	X, Y, W, H     float32
	Color          Color
	Asset          Asset
	U0, V0, U1, V1 float32
	Font           FontRef
	Px             int
	First, N       int
}

// DrawList is a frame's ops in paint order, later on top. Reset keeps the
// backing arrays, so a list reused every frame stops allocating.
type DrawList struct {
	Ops    []Op
	Glyphs []Glyph
}

// Reset empties the list and keeps its capacity.
func (l *DrawList) Reset() {
	l.Ops = l.Ops[:0]
	l.Glyphs = l.Glyphs[:0]
}
