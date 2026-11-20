// Package bmfont turns MUGEN / Ikemen GO bitmap fonts (.fnt v1, and v2
// .def + .sff) into colored glyph images and lays text out as Ikemen GO's
// Fnt.DrawText does (src/font.go, commit 54d4148a). It has no rendering
// dependency: the renderer uploads the glyphs and draws the placements.
package bmfont

import (
	"bytes"
	"fmt"
	"image/color"
	"io/fs"
	"math"
	"path"
	"strings"

	"github.com/openkakutou/fnt"
	"github.com/openkakutou/sff"
)

// Glyph is one character in one bank: W x H colored pixels, transparent
// where the font is, and the sprite axis subtracted when drawing.
type Glyph struct {
	Pixels     []color.RGBA
	W, H       int
	OffX, OffY int
}

// Font is a loaded bitmap font.
type Font struct {
	Size, Spacing, Offset [2]int
	// banks[b][r] is the glyph of r in bank b; bank 0 always exists.
	banks []map[rune]*Glyph
}

// Placed is one glyph to draw: its top-left corner, before scaling the
// glyph by the layout's scale.
type Placed struct {
	Glyph *Glyph
	X, Y  float64
}

// Load reads the font name in fsys: a .fnt file is a v1 font, anything else
// a v2 .def whose File is read beside it.
func Load(fsys fs.FS, name string) (*Font, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(path.Ext(name), ".fnt") {
		return loadV1(data)
	}
	return loadV2(fsys, name, data)
}

// loadV1 builds a glyph map per palette bank, as loadFntV1: with
// c = min(255, ceil(colors/16)*16), bank i keeps the first 256-c colors and
// takes its last c colors from the c-color block i places from the end.
func loadV1(data []byte) (*Font, error) {
	src, err := fnt.Parse(data)
	if err != nil {
		return nil, err
	}
	f := &Font{Size: pt(src.Size), Spacing: pt(src.Spacing), Offset: pt(src.Offset)}
	c := min(255, int(math.Ceil(float64(src.Colors)/16))*16)
	img := src.Image
	for i := 0; i < 255/c; i++ {
		var pal [256]fnt.RGB
		copy(pal[:256-c], img.Palette[:256-c])
		copy(pal[256-c:], img.Palette[256-c*(i+1):256-c*i])
		bank := map[rune]*Glyph{}
		for r, g := range src.Glyphs {
			bank[r] = cutV1(img, pal, g)
		}
		f.banks = append(f.banks, bank)
	}
	if len(f.banks) == 0 {
		return nil, fmt.Errorf("bmfont: no palette bank")
	}
	return f, nil
}

func cutV1(img fnt.Image, pal [256]fnt.RGB, g fnt.Glyph) *Glyph {
	out := &Glyph{W: g.Width, H: img.Height, Pixels: make([]color.RGBA, g.Width*img.Height)}
	for y := 0; y < img.Height; y++ {
		for x := 0; x < g.Width && g.X+x < img.Width; x++ {
			idx := img.Pixels[y*img.Width+g.X+x]
			if idx == 0 {
				continue // index 0 is transparent
			}
			p := pal[idx]
			out.Pixels[y*g.Width+x] = color.RGBA{p.R, p.G, p.B, 255}
		}
	}
	return out
}

// loadV2 reads the sprites of the .def's File (beside the .def), as
// LoadFntSff: group 0 holds the glyphs (every group with BankType sprite,
// the bank then being the group), and with BankType palette bank i draws
// with the sff palette 0,i.
func loadV2(fsys fs.FS, name string, def []byte) (*Font, error) {
	if head, err := fnt.ParseV2(def, nil); err == nil {
		if head.Type == "truetype" {
			return nil, fmt.Errorf("bmfont: %s is a TrueType font, not supported", name)
		}
		return nil, fmt.Errorf("bmfont: %s names no glyph file", name)
	}
	sffName := path.Join(path.Dir(name), v2File(def))
	sffData, err := fs.ReadFile(fsys, sffName)
	if err != nil {
		return nil, fmt.Errorf("bmfont: sprites of %s: %w", name, err)
	}
	r := bytes.NewReader(sffData)
	src, err := fnt.ParseV2(def, r)
	if err != nil {
		return nil, err
	}
	f := &Font{Size: pt(src.Size), Spacing: pt(src.Spacing), Offset: pt(src.Offset)}
	pals := v2Palettes(sffData)
	if src.BankType == "sprite" {
		maxBank := 0
		for b := range src.Banks {
			maxBank = max(maxBank, b)
		}
		f.banks = make([]map[rune]*Glyph, maxBank+1)
		for b, glyphs := range src.Banks {
			f.banks[b] = cutV2(r, glyphs, src, nil)
		}
		if f.banks[0] == nil {
			f.banks[0] = map[rune]*Glyph{}
		}
		return f, nil
	}
	glyphs := src.Banks[0]
	if len(pals) == 0 {
		f.banks = []map[rune]*Glyph{cutV2(r, glyphs, src, nil)}
		return f, nil
	}
	for i := range pals {
		f.banks = append(f.banks, cutV2(r, glyphs, src, &pals[i]))
	}
	return f, nil
}

// v2File is the last File key of a v2 .def, the one fnt.ParseV2 reads the
// sprites of.
func v2File(def []byte) string {
	file := ""
	for _, line := range strings.Split(string(def), "\n") {
		line, _, _ = strings.Cut(line, ";")
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), "file") {
			file = strings.Trim(strings.TrimSpace(v), `"`)
		}
	}
	return file
}

// v2Palettes returns the palettes 0,0, 0,1, ... of an sff v2, in order,
// stopping at the first missing one; nil for an sff v1.
func v2Palettes(data []byte) []sff.Palette {
	r := bytes.NewReader(data)
	table, err := sff.ParseV2(r)
	if err != nil {
		return nil
	}
	var out []sff.Palette
	for n := 0; ; n++ {
		idx := -1
		for i, p := range table.Palettes {
			if p.Group == 0 && p.Number == n {
				idx = i
			}
		}
		if idx < 0 {
			return out
		}
		pal, err := sff.ResolveV2Palette(table, r, idx, nil)
		if err != nil {
			return out
		}
		out = append(out, pal)
	}
}

func cutV2(r *bytes.Reader, glyphs map[rune]fnt.GlyphV2, src *fnt.FontV2, pal *sff.Palette) map[rune]*Glyph {
	out := map[rune]*Glyph{}
	axisY := map[[2]int]int{}
	for _, g := range src.Sprites {
		for _, s := range g.Sprites {
			axisY[[2]int{s.Group, s.Image}] = s.AxisY
		}
	}
	for ch, g := range glyphs {
		px, w, h, err := sff.ResolveSpritePixels(r, g.Group, g.Image, pal)
		if err != nil || w == 0 || h == 0 {
			continue
		}
		out[ch] = &Glyph{Pixels: px, W: w, H: h, OffX: g.AxisX, OffY: axisY[[2]int{g.Group, g.Image}]}
	}
	return out
}

func pt(p fnt.Point) [2]int { return [2]int{p.X, p.Y} }

// bank returns bank b's glyphs, bank 0 when b is out of range (DrawText).
func (f *Font) bank(b int) map[rune]*Glyph {
	if b < 0 || b >= len(f.banks) || f.banks[b] == nil {
		return f.banks[0]
	}
	return f.banks[b]
}

// Glyph returns the glyph of r in bank b, nil when the font lacks it.
func (f *Font) Glyph(b int, r rune) *Glyph { return f.bank(b)[r] }

// charWidth is CharWidth: a space is Size wide, an unknown character 0.
func (f *Font) charWidth(glyphs map[rune]*Glyph, r rune) int {
	if r == ' ' {
		return f.Size[0]
	}
	if g := glyphs[r]; g != nil {
		return g.W
	}
	return 0
}

// TextWidth is Ikemen GO's Fnt.TextWidth: glyph widths plus the spacing
// between them, unknown characters counted as spaces as DrawText does.
func (f *Font) TextWidth(text string, b int) int {
	glyphs := f.bank(b)
	runes := []rune(text)
	w := 0
	for i, r := range runes {
		if glyphs[r] == nil {
			r = ' '
		}
		cw := f.charWidth(glyphs, r)
		if cw+f.Spacing[0] <= 0 {
			continue
		}
		w += cw
		if i < len(runes)-1 {
			w += f.Spacing[0]
		}
	}
	return w
}

// Layout appends to dst the glyphs of text printed at (x, y) in bank b,
// align 1 left, 0 centered, -1 right, at scale, as DrawText places them:
// the font Offset is added, y is the baseline (the glyphs' top is
// Size.y-1 above it), unknown characters are spaces.
func (f *Font) Layout(text string, b, align int, x, y, scale float64, dst []Placed) []Placed {
	dst = dst[:0]
	if text == "" || scale == 0 {
		return dst
	}
	glyphs := f.bank(b)
	x += float64(f.Offset[0]) * scale
	y += float64(f.Offset[1]-f.Size[1]+1) * scale
	switch {
	case align == 0:
		x -= float64(f.TextWidth(text, b)) * scale * 0.5
	case align < 0:
		x -= float64(f.TextWidth(text, b)) * scale
	}
	for _, r := range text {
		g := glyphs[r]
		advance := float64(f.Size[0])
		if g != nil {
			dst = append(dst, Placed{Glyph: g, X: x - float64(g.OffX)*scale, Y: y - float64(g.OffY)*scale})
			advance = float64(g.W)
		}
		x += (advance + float64(f.Spacing[0])) * scale
	}
	return dst
}
