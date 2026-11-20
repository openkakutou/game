package gfx

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/openkakutou/game/native/internal/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// atlasSize is the side of a glyph atlas page, in pixels.
const atlasSize = 1024

type atlasKey struct {
	font ui.FontRef
	px   int
}

// glyphEntry is where a glyph sits in its atlas page, and where its bitmap
// goes relative to the pen on the baseline.
type glyphEntry struct {
	page           int
	u0, v0, u1, v1 float32
	dx, dy, w, h   float32
}

// atlasPage is one alpha texture of an atlas and its free room.
type atlasPage struct {
	tex    *Texture
	packer shelfPacker
}

func newAtlasPage() *atlasPage {
	var id uint32
	gl.GenTextures(1, &id)
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.ALPHA, atlasSize, atlasSize, 0, gl.ALPHA, gl.UNSIGNED_BYTE, nil)
	return &atlasPage{tex: &Texture{id: id, W: atlasSize, H: atlasSize}, packer: shelfPacker{size: atlasSize}}
}

// glyphAtlas holds the rasterized glyphs of one font at one pixel size, on as
// many alpha pages as they need: a glyph is rasterized the first time it is
// drawn, and every one gets a place.
type glyphAtlas struct {
	face   font.Face
	pages  []*atlasPage
	glyphs map[rune]glyphEntry
}

func newGlyphAtlas(face font.Face) *glyphAtlas {
	return &glyphAtlas{face: face, glyphs: map[rune]glyphEntry{}}
}

// add rasterizes r onto the first page with room, or a new page. A glyph
// larger than a whole page (a font size of thousands of pixels) is reported
// once on stderr and left out.
func (a *glyphAtlas) add(r rune) glyphEntry {
	dr, mask, maskp, _, ok := a.face.Glyph(fixed.Point26_6{}, r)
	alpha, isAlpha := mask.(*image.Alpha)
	w, h := dr.Dx(), dr.Dy()
	if !ok || !isAlpha || w == 0 || h == 0 {
		a.glyphs[r] = glyphEntry{}
		return glyphEntry{}
	}
	if w > atlasSize || h > atlasSize {
		fmt.Fprintf(os.Stderr, "openkakutou: the glyph %q is %dx%d px, larger than a %d px atlas page: not drawn\n", r, w, h, atlasSize)
		a.glyphs[r] = glyphEntry{}
		return glyphEntry{}
	}
	page, x, y := a.room(w, h)
	bitmap := make([]byte, w*h)
	for row := 0; row < h; row++ {
		from := (row+maskp.Y-alpha.Rect.Min.Y)*alpha.Stride + maskp.X - alpha.Rect.Min.X
		copy(bitmap[row*w:(row+1)*w], alpha.Pix[from:from+w])
	}
	gl.BindTexture(gl.TEXTURE_2D, a.pages[page].tex.id)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(x), int32(y), int32(w), int32(h), gl.ALPHA, gl.UNSIGNED_BYTE, unsafe.Pointer(&bitmap[0]))
	e := glyphEntry{
		page: page,
		u0:   float32(x) / atlasSize, v0: float32(y) / atlasSize,
		u1: float32(x+w) / atlasSize, v1: float32(y+h) / atlasSize,
		dx: float32(dr.Min.X), dy: float32(dr.Min.Y), w: float32(w), h: float32(h),
	}
	a.glyphs[r] = e
	return e
}

// room reserves a w x h place (no larger than a page) and returns its page
// and corner, adding a page when none has room.
func (a *glyphAtlas) room(w, h int) (page, x, y int) {
	for i, p := range a.pages {
		if x, y, ok := p.packer.place(w, h); ok {
			return i, x, y
		}
	}
	p := newAtlasPage()
	a.pages = append(a.pages, p)
	x, y, _ = p.packer.place(w, h) // an empty page holds any glyph that passed the size check
	return len(a.pages) - 1, x, y
}

func (a *glyphAtlas) glyph(r rune) glyphEntry {
	if e, ok := a.glyphs[r]; ok {
		return e
	}
	return a.add(r)
}

// listPainter executes a ui.DrawList: image textures and glyph atlases are
// created the first time they are used and kept, and consecutive quads of
// one texture go out in one draw call, in the list's order.
type listPainter struct {
	win       *Window
	pack      *ui.Pack
	portraits *portraitCache
	textures  map[ui.Asset]*Texture
	atlases   map[atlasKey]*glyphAtlas
	// batch holds textured vertices (x, y, u, v, r, g, b, a) of batchTex.
	batch       []float32
	batchTex    *Texture
	batchRepeat [2]bool // wrap each axis instead of clamping, for tiled images
}

func newListPainter(win *Window, pack *ui.Pack, portraits *portraitCache) *listPainter {
	return &listPainter{
		win: win, pack: pack, portraits: portraits,
		textures: map[ui.Asset]*Texture{},
		atlases:  map[atlasKey]*glyphAtlas{},
		batch:    make([]float32, 0, 8*6*1024),
	}
}

// release deletes every texture and glyph atlas page the painter holds.
func (p *listPainter) release() {
	for _, tex := range p.textures {
		if tex != nil {
			tex.Delete()
		}
	}
	for _, atlas := range p.atlases {
		for _, page := range atlas.pages {
			page.tex.Delete()
		}
	}
	clear(p.textures)
	clear(p.atlases)
}

// draw paints the list over what is already queued.
func (p *listPainter) draw(list *ui.DrawList) {
	for i := range list.Ops {
		op := &list.Ops[i]
		switch op.Kind {
		case ui.OpRect:
			p.flushBatch()
			p.win.Rect(op.X, op.Y, op.W, op.H, Color(op.Color))
		case ui.OpImage:
			if tex := p.texture(op.Asset); tex != nil {
				repeat := [2]bool{op.U1 > 1, op.V1 > 1}
				p.quad(tex, repeat, op.X, op.Y, op.X+op.W, op.Y+op.H, op.U0, op.V0, op.U1, op.V1, op.Color)
			}
		case ui.OpPortrait:
			if tex := p.portraits.texture(op.Portrait); tex != nil {
				p.quad(tex, [2]bool{}, op.X, op.Y, op.X+op.W, op.Y+op.H, op.U0, op.V0, op.U1, op.V1, op.Color)
			}
		case ui.OpText:
			p.text(list, op)
		}
	}
	p.flushBatch()
}

func (p *listPainter) text(list *ui.DrawList, op *ui.Op) {
	key := atlasKey{op.Font, op.Px}
	atlas := p.atlases[key]
	if atlas == nil {
		face := p.pack.Face(op.Font, op.Px)
		if face == nil {
			return
		}
		atlas = newGlyphAtlas(face)
		p.atlases[key] = atlas
	}
	baseline := float32(math.Round(float64(op.Y)))
	for _, g := range list.Glyphs[op.First : op.First+op.N] {
		e := atlas.glyph(g.R)
		if e.w == 0 {
			continue
		}
		x := float32(math.Round(float64(op.X+g.X))) + e.dx
		y := baseline + e.dy
		p.quad(atlas.pages[e.page].tex, [2]bool{}, x, y, x+e.w, y+e.h, e.u0, e.v0, e.u1, e.v1, op.Color)
	}
}

// quad queues a textured rectangle, after flushing what another texture or
// the colored batch still holds, to keep the draw order.
func (p *listPainter) quad(tex *Texture, repeat [2]bool, x0, y0, x1, y1, u0, v0, u1, v1 float32, c ui.Color) {
	if p.batchTex != tex || p.batchRepeat != repeat {
		p.flushBatch()
		p.batchTex, p.batchRepeat = tex, repeat
	}
	p.win.Flush()
	p.batch = append(p.batch,
		x0, y0, u0, v0, c.R, c.G, c.B, c.A, x1, y0, u1, v0, c.R, c.G, c.B, c.A, x1, y1, u1, v1, c.R, c.G, c.B, c.A,
		x0, y0, u0, v0, c.R, c.G, c.B, c.A, x1, y1, u1, v1, c.R, c.G, c.B, c.A, x0, y1, u0, v1, c.R, c.G, c.B, c.A)
}

// flushBatch draws the queued textured vertices in one call.
func (p *listPainter) flushBatch() {
	if len(p.batch) == 0 {
		return
	}
	const stride = 8 * 4
	gl.Enable(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, p.batchTex.id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, wrapMode(p.batchRepeat[0]))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, wrapMode(p.batchRepeat[1]))
	gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
	gl.VertexPointer(2, gl.FLOAT, stride, unsafe.Pointer(&p.batch[0]))
	gl.TexCoordPointer(2, gl.FLOAT, stride, unsafe.Pointer(&p.batch[2]))
	gl.ColorPointer(4, gl.FLOAT, stride, unsafe.Pointer(&p.batch[4]))
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(p.batch)/8))
	gl.DisableClientState(gl.TEXTURE_COORD_ARRAY)
	gl.Disable(gl.TEXTURE_2D)
	p.batch = p.batch[:0]
}

// wrapMode is how a texture axis wraps: repeating for a tiled image, else
// clamped so a scaled image or sprite frame never bleeds across its edges.
func wrapMode(repeat bool) int32 {
	if repeat {
		return gl.REPEAT
	}
	return gl.CLAMP_TO_EDGE
}

// texture returns an asset's texture, decoded and uploaded the first time.
// An asset that cannot be read is reported once and then skipped.
func (p *listPainter) texture(a ui.Asset) *Texture {
	if tex, seen := p.textures[a]; seen {
		return tex
	}
	tex, err := p.load(a)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openkakutou: screenpack image %s: %v\n", a.Path, err)
	}
	p.textures[a] = tex
	return tex
}

func (p *listPainter) load(a ui.Asset) (*Texture, error) {
	file, err := p.pack.Open(a)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	straight := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(straight, straight.Bounds(), img, b.Min, draw.Src)
	px := make([]color.RGBA, b.Dx()*b.Dy())
	for i := range px {
		px[i] = color.RGBA{straight.Pix[4*i], straight.Pix[4*i+1], straight.Pix[4*i+2], straight.Pix[4*i+3]}
	}
	tex := NewTexture(px, b.Dx(), b.Dy())
	tex.smooth()
	return tex, nil
}
