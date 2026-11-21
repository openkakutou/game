// Package gfx is the native renderer: a raw go-gl (OpenGL 2.1) + SDL2 window
// that draws colored quads, textured sprites and bitmap text. It draws what
// the game package describes and holds no game rules. See the project's
// "Native rendering performance constraints": the batch buffer below is
// reused every frame (no per-frame allocation) and all colored geometry
// goes out in a single draw call.
package gfx

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"unicode"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/veandco/go-sdl2/sdl"
)

// Virtual screen size every screen is laid out in; scaled to the window.
const (
	VW = 320
	VH = 240
)

// Color is an RGBA color with 0..1 components.
type Color struct{ R, G, B, A float32 }

// Rgb builds an opaque color from 0..255 components.
func Rgb(r, g, b uint8) Color { return Color{float32(r) / 255, float32(g) / 255, float32(b) / 255, 1} }

// Texture is an uploaded sprite image.
type Texture struct {
	id   uint32
	W, H int
}

// Window owns the SDL window and its OpenGL context.
type Window struct {
	win  *sdl.Window
	ctx  sdl.GLContext
	W, H int32
	// view is the letterboxed VWxVH viewport in window pixels (x, y, w, h).
	view  [4]int32
	verts []float32
	// shiftY is the vertical offset BeginShift applies, which Clip follows.
	shiftY float32
	// round is RoundRect's reused triangle buffer.
	round  []float32
	nverts int
}

// NewWindow opens a window (hidden if headless is true is not needed:
// Xvfb provides a display) with an OpenGL 2.1 context; ApplyVideo sets vsync and fullscreen.
func NewWindow(title string, w, h int32) (*Window, error) {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return nil, fmt.Errorf("initializing SDL: %w", err)
	}
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 2)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 1)
	win, err := sdl.CreateWindow(title, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, w, h,
		sdl.WINDOW_OPENGL|sdl.WINDOW_RESIZABLE)
	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("creating window: %w", err)
	}
	ctx, err := win.GLCreateContext()
	if err != nil {
		win.Destroy()
		sdl.Quit()
		return nil, fmt.Errorf("creating OpenGL context: %w", err)
	}
	if err := gl.Init(); err != nil {
		return nil, fmt.Errorf("loading OpenGL: %w", err)
	}
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.EnableClientState(gl.VERTEX_ARRAY)
	gl.EnableClientState(gl.COLOR_ARRAY)
	return &Window{win: win, ctx: ctx, W: w, H: h, verts: make([]float32, 0, 6*6*4096)}, nil
}

// ApplyVideo applies the video settings: fullscreen (the desktop's own
// resolution), vsync, and, when windowed, the window size.
func (w *Window) ApplyVideo(fullscreen, vsync bool, width, height int32) error {
	var flags uint32
	if fullscreen {
		flags = sdl.WINDOW_FULLSCREEN_DESKTOP
	}
	// Only what differs is applied: a needless change re-maps the window.
	if w.win.GetFlags()&sdl.WINDOW_FULLSCREEN_DESKTOP != flags {
		if err := w.win.SetFullscreen(flags); err != nil {
			return fmt.Errorf("changing fullscreen: %w", err)
		}
	}
	if curW, curH := w.win.GetSize(); !fullscreen && (curW != width || curH != height) {
		w.win.SetSize(width, height)
	}
	interval := 0
	if vsync {
		interval = 1
	}
	if err := sdl.GLSetSwapInterval(interval); err != nil {
		return fmt.Errorf("changing vsync: %w", err)
	}
	return nil
}

// Close releases the window and SDL.
func (w *Window) Close() {
	sdl.GLDeleteContext(w.ctx)
	w.win.Destroy()
	sdl.Quit()
}

// Renderer returns the OpenGL renderer string (useful to log).
func (w *Window) Renderer() string { return gl.GoStr(gl.GetString(gl.RENDERER)) }

// BeginFrame clears the screen and sets a VWxVH virtual coordinate space,
// letterboxed to keep the 4:3 aspect ratio.
func (w *Window) BeginFrame() {
	dw, dh := w.win.GLGetDrawableSize()
	w.W, w.H = dw, dh
	vw, vh := dw, dw*VH/VW
	if vh > dh {
		vh, vw = dh, dh*VW/VH
	}
	gl.Viewport(0, 0, dw, dh)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	w.view = [4]int32{(dw - vw) / 2, (dh - vh) / 2, vw, vh}
	w.useVirtualSpace()
	w.nverts = 0
	w.verts = w.verts[:0]
}

// useVirtualSpace draws in the letterboxed VWxVH space.
func (w *Window) useVirtualSpace() {
	gl.Viewport(w.view[0], w.view[1], w.view[2], w.view[3])
	w.setOrtho(VW, VH)
}

func (w *Window) setOrtho(width, height float64) {
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Ortho(0, width, height, 0, -1, 1)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
}

// BeginPixels switches to window pixels over the whole window, for screens
// laid out for any aspect ratio, and returns its size; EndPixels goes back
// to the virtual space. Queued geometry is flushed at each switch.
func (w *Window) BeginPixels() (width, height int) {
	w.Flush()
	gl.Viewport(0, 0, w.W, w.H)
	w.setOrtho(float64(w.W), float64(w.H))
	return int(w.W), int(w.H)
}

// EndPixels leaves the window pixel space.
func (w *Window) EndPixels() {
	w.Flush()
	w.useVirtualSpace()
}

// Flush draws every batched colored quad in one call.
func (w *Window) Flush() {
	if w.nverts == 0 {
		return
	}
	gl.Disable(gl.TEXTURE_2D)
	gl.VertexPointer(2, gl.FLOAT, 6*4, unsafe.Pointer(&w.verts[0]))
	gl.ColorPointer(4, gl.FLOAT, 6*4, unsafe.Pointer(&w.verts[2]))
	gl.DrawArrays(gl.TRIANGLES, 0, int32(w.nverts))
	w.nverts = 0
	w.verts = w.verts[:0]
}

// EndFrame flushes pending geometry and presents the frame.
func (w *Window) EndFrame() {
	w.Flush()
	w.win.GLSwap()
}

// Rect queues a filled rectangle in virtual coordinates.
func (w *Window) Rect(x, y, wd, ht float32, c Color) {
	x2, y2 := x+wd, y+ht
	w.verts = append(w.verts,
		x, y, c.R, c.G, c.B, c.A, x2, y, c.R, c.G, c.B, c.A, x2, y2, c.R, c.G, c.B, c.A,
		x, y, c.R, c.G, c.B, c.A, x2, y2, c.R, c.G, c.B, c.A, x, y2, c.R, c.G, c.B, c.A)
	w.nverts += 6
}

// Text queues s at (x, y) in virtual coordinates; scale is the pixel size of one font pixel.
func (w *Window) Text(x, y, scale float32, c Color, s string) {
	cx := x
	for _, r := range s {
		if rows, ok := glyphRows[unicode.ToUpper(r)]; ok {
			for ry, bits := range rows {
				for col := 0; col < 5; col++ {
					if bits&(1<<(4-col)) != 0 {
						w.Rect(cx+float32(col)*scale, y+float32(ry)*scale, scale, scale, c)
					}
				}
			}
		}
		cx += GlyphWidth * scale
	}
}

// TextCentered draws s horizontally centered on the virtual screen.
func (w *Window) TextCentered(y, scale float32, c Color, s string) {
	w.Text((VW-float32(TextWidth(s))*scale)/2, y, scale, c, s)
}

// NewTexture uploads RGBA pixels as a nearest-filtered texture.
func NewTexture(px []color.RGBA, width, height int) *Texture {
	var id uint32
	gl.GenTextures(1, &id)
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&px[0]))
	return &Texture{id: id, W: width, H: height}
}

// smooth makes the texture linearly filtered, as screenpack images are
// scaled to any size.
func (t *Texture) smooth() {
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
}

// Delete frees the texture.
func (t *Texture) Delete() { gl.DeleteTextures(1, &t.id) }

// Sprite draws tex with its top-left at (x, y), optionally mirrored horizontally.
// Pending colored quads are flushed first so draw order is preserved.
func (w *Window) Sprite(t *Texture, x, y float32, flip bool) {
	w.SpriteSized(t, x, y, float32(t.W), float32(t.H), flip)
}

// SpriteSized draws t with its top-left at x,y stretched to wd x ht.
func (w *Window) SpriteSized(t *Texture, x, y, wd, ht float32, flip bool) {
	w.Flush()
	gl.Enable(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.Color4f(1, 1, 1, 1)
	gl.DisableClientState(gl.COLOR_ARRAY)
	u0, u1 := float32(0), float32(1)
	if flip {
		u0, u1 = 1, 0
	}
	x2, y2 := x+wd, y+ht
	gl.Begin(gl.QUADS)
	gl.TexCoord2f(u0, 0)
	gl.Vertex2f(x, y)
	gl.TexCoord2f(u1, 0)
	gl.Vertex2f(x2, y)
	gl.TexCoord2f(u1, 1)
	gl.Vertex2f(x2, y2)
	gl.TexCoord2f(u0, 1)
	gl.Vertex2f(x, y2)
	gl.End()
	gl.EnableClientState(gl.COLOR_ARRAY)
	gl.Disable(gl.TEXTURE_2D)
}

// SpriteTinted is Sprite with its colours multiplied by mul and add added
// where it is opaque (a PalFX): a second, additive pass paints the add colour
// through the sprite's alpha.
func (w *Window) SpriteTinted(t *Texture, x, y float32, flip, additive bool, mul, add [3]float32) {
	w.SpriteTintedSized(t, x, y, float32(t.W), float32(t.H), flip, additive, mul, add)
}

// SpriteTintedSized is SpriteTinted stretched to wd x ht.
func (w *Window) SpriteTintedSized(t *Texture, x, y, wd, ht float32, flip, additive bool, mul, add [3]float32) {
	w.Flush()
	if additive {
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
		defer gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	}
	gl.Enable(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.DisableClientState(gl.COLOR_ARRAY)
	u0, u1 := float32(0), float32(1)
	if flip {
		u0, u1 = 1, 0
	}
	quad := func() {
		x2, y2 := x+wd, y+ht
		gl.Begin(gl.QUADS)
		gl.TexCoord2f(u0, 0)
		gl.Vertex2f(x, y)
		gl.TexCoord2f(u1, 0)
		gl.Vertex2f(x2, y)
		gl.TexCoord2f(u1, 1)
		gl.Vertex2f(x2, y2)
		gl.TexCoord2f(u0, 1)
		gl.Vertex2f(x, y2)
		gl.End()
	}
	gl.Color4f(mul[0], mul[1], mul[2], 1)
	quad()
	if add != [3]float32{} {
		gl.TexEnvi(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.COMBINE)
		gl.TexEnvi(gl.TEXTURE_ENV, gl.COMBINE_RGB, gl.REPLACE)
		gl.TexEnvi(gl.TEXTURE_ENV, gl.SOURCE0_RGB, gl.PRIMARY_COLOR)
		gl.TexEnvi(gl.TEXTURE_ENV, gl.COMBINE_ALPHA, gl.REPLACE)
		gl.TexEnvi(gl.TEXTURE_ENV, gl.SOURCE0_ALPHA, gl.TEXTURE)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
		gl.Color4f(add[0], add[1], add[2], 1)
		quad()
		if !additive {
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		}
		gl.TexEnvi(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
	}
	gl.Color4f(1, 1, 1, 1)
	gl.EnableClientState(gl.COLOR_ARRAY)
	gl.Disable(gl.TEXTURE_2D)
}

// Blend is how a sprite mixes with what is behind it.
type Blend int

const (
	BlendNormal Blend = iota
	BlendAdd
	BlendSub
)

// SpriteBlended is SpriteSized drawn with blend, its colours weighted by
// alpha (0..1).
func (w *Window) SpriteBlended(t *Texture, x, y, wd, ht float32, flip bool, blend Blend, alpha float32) {
	w.SpriteRotated(t, x, y, wd, ht, flip, blend, alpha, 0, 0, 0)
}

// SpriteRotated is SpriteBlended turned deg degrees counter-clockwise on
// screen around px, py.
func (w *Window) SpriteRotated(t *Texture, x, y, wd, ht float32, flip bool, blend Blend, alpha, px, py, deg float32) {
	w.Flush()
	if deg != 0 {
		gl.PushMatrix()
		gl.Translatef(px, py, 0)
		gl.Rotatef(-deg, 0, 0, 1) // y points down: a negative turn is counter-clockwise
		gl.Translatef(-px, -py, 0)
		defer gl.PopMatrix()
	}
	switch blend {
	case BlendAdd:
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
	case BlendSub:
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
		gl.BlendEquation(gl.FUNC_REVERSE_SUBTRACT)
	}
	gl.Enable(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.Color4f(1, 1, 1, alpha)
	gl.DisableClientState(gl.COLOR_ARRAY)
	u0, u1 := float32(0), float32(1)
	if flip {
		u0, u1 = 1, 0
	}
	x2, y2 := x+wd, y+ht
	gl.Begin(gl.QUADS)
	gl.TexCoord2f(u0, 0)
	gl.Vertex2f(x, y)
	gl.TexCoord2f(u1, 0)
	gl.Vertex2f(x2, y)
	gl.TexCoord2f(u1, 1)
	gl.Vertex2f(x2, y2)
	gl.TexCoord2f(u0, 1)
	gl.Vertex2f(x, y2)
	gl.End()
	gl.EnableClientState(gl.COLOR_ARRAY)
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4f(1, 1, 1, 1)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
}

// SpriteAdd is Sprite drawn additively: the sprite's colours are added to
// what is behind it, as an .air frame's A blend asks.
func (w *Window) SpriteAdd(t *Texture, x, y float32, flip bool) {
	w.Flush()
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
	w.Sprite(t, x, y, flip)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
}

// additiveBlend reports whether an .air frame's blend adds the sprite (A,
// A1, AS…D…).
// shortcut: the source/destination weights of AS…D… and the S
// (subtractive) blend are not applied; upgrade with the shader renderer
// (decision 028).
func additiveBlend(blend string) bool {
	b := strings.TrimSpace(blend)
	return b != "" && (b[0] == 'A' || b[0] == 'a')
}

// Clip limits drawing to the virtual rectangle x, y, wd, ht until NoClip.
func (w *Window) Clip(x, y, wd, ht float32) {
	w.Flush()
	y += w.shiftY
	sx := float32(w.view[2]) / VW
	sy := float32(w.view[3]) / VH
	px := w.view[0] + int32(x*sx)
	py := w.view[1] + w.view[3] - int32((y+ht)*sy)
	gl.Enable(gl.SCISSOR_TEST)
	gl.Scissor(px, py, int32(max(wd, 0)*sx), int32(max(ht, 0)*sy))
}

// NoClip lifts Clip.
func (w *Window) NoClip() {
	w.Flush()
	gl.Disable(gl.SCISSOR_TEST)
}

// SavePNG writes the current framebuffer to path (used for verification).
func (w *Window) SavePNG(path string) error {
	w.Flush()
	pix := make([]byte, int(w.W)*int(w.H)*4)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadPixels(0, 0, w.W, w.H, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&pix[0]))
	img := image.NewRGBA(image.Rect(0, 0, int(w.W), int(w.H)))
	stride := int(w.W) * 4
	for y := 0; y < int(w.H); y++ {
		copy(img.Pix[y*stride:(y+1)*stride], pix[(int(w.H)-1-y)*stride:(int(w.H)-y)*stride])
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if !strings.HasSuffix(path, ".png") {
		return fmt.Errorf("capture path must end in .png")
	}
	return png.Encode(f, img)
}

// BeginShift moves everything drawn until EndShift down by dy virtual
// pixels, clips included.
func (w *Window) BeginShift(dy float32) {
	w.Flush()
	gl.PushMatrix()
	gl.Translatef(0, dy, 0)
	w.shiftY = dy
}

// EndShift ends BeginShift.
func (w *Window) EndShift() {
	w.Flush()
	gl.PopMatrix()
	w.shiftY = 0
}
