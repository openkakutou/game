package ui_test

import (
	"bytes"
	"image"
	"image/png"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/ui"
	"github.com/openkakutou/screenpack"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var view720 = ui.View{W: 1280, H: 720}

const rootJSON = `{
  "formatVersion": 1, "name": "Test", "author": "Test", "version": "1.0.0",
  "fonts": { "body": "fonts/NotoSans-Regular.ttf" },
  "sounds": { "move": "s.ogg", "confirm": "s.ogg", "cancel": "s.ogg", "error": "s.ogg" },
  "textStyles": {
    "default": { "font": "body", "size": 5, "color": "#ffffff" },
    "left":    { "font": "body", "size": 5, "color": "#ffffff", "align": "left" },
    "center":  { "font": "body", "size": 5, "color": "#ffffff", "align": "center" },
    "right":   { "font": "body", "size": 5, "color": "#ffffff", "align": "right" },
    "upper":   { "font": "body", "size": 5, "color": "#ffffff", "uppercase": true },
    "fancy":   { "font": "body", "size": 5, "color": "#00ff00",
                 "outline": { "color": "#0000ff", "width": 0.5 },
                 "shadow": { "color": "#00000080", "offset": [1, 1] } }
  }
}`

// fontBytes is the font the test packs use, read from the built-in screenpack.
func fontBytes(t *testing.T) []byte {
	t.Helper()
	data, err := fs.ReadFile(screenpack.DefaultFS(), "fonts/NotoSans-Regular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// pngOf encodes a w x h opaque image.
func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// gameWith is a game folder whose screenpacks/default has the test root file,
// the test font, a title screen made of parts, and extra files.
func gameWith(t *testing.T, parts string, extra fstest.MapFS) fstest.MapFS {
	t.Helper()
	game := fstest.MapFS{
		"screenpacks/default/screenpack.json":            {Data: []byte(rootJSON)},
		"screenpacks/default/fonts/NotoSans-Regular.ttf": {Data: fontBytes(t)},
		"screenpacks/default/screens/title.json":         {Data: []byte(`{"formatVersion": 1, ` + parts + `}`)},
	}
	game["screenpacks/default/s.ogg"] = &fstest.MapFile{Data: []byte("sound")}
	for name, file := range extra {
		game["screenpacks/default/"+name] = file
	}
	return game
}

// layoutOf loads the game folder's pack, which must be valid.
func layoutOf(t *testing.T, game fs.FS) *ui.Layout {
	t.Helper()
	pack, notices := ui.LoadPack(game)
	if len(notices) > 0 {
		t.Fatalf("pack refused: %v", notices)
	}
	return ui.NewLayout(pack)
}

func menuState(entries []string, cursor int) ui.TitleState {
	return ui.TitleState{Phase: ui.PhaseMenu, Entries: entries, Cursor: cursor, ElapsedMs: 10000}
}

func draw(l *ui.Layout, s ui.TitleState) *ui.DrawList {
	var list ui.DrawList
	l.Title(&list, view720, s)
	return &list
}

// textOf is the characters of a text op.
func textOf(l *ui.DrawList, op ui.Op) string {
	var runes []rune
	for _, g := range l.Glyphs[op.First : op.First+op.N] {
		runes = append(runes, g.R)
	}
	return string(runes)
}

// textOps are the text ops whose string is s.
func textOps(l *ui.DrawList, s string) []ui.Op {
	var out []ui.Op
	for _, op := range l.Ops {
		if op.Kind == ui.OpText && textOf(l, op) == s {
			out = append(out, op)
		}
	}
	return out
}

// onlyTexts are all the strings drawn, in order.
func onlyTexts(l *ui.DrawList) []string {
	var out []string
	for _, op := range l.Ops {
		if op.Kind == ui.OpText {
			out = append(out, textOf(l, op))
		}
	}
	return out
}

// rectOps are the filled rectangles of a color.
func rectOps(l *ui.DrawList, c ui.Color) []ui.Op {
	var out []ui.Op
	for _, op := range l.Ops {
		if op.Kind == ui.OpRect && op.Color == c {
			out = append(out, op)
		}
	}
	return out
}

// imageOps are the image quads of an asset of the pack under test (not the
// ones of parts it borrows from the default).
func imageOps(l *ui.DrawList, path string) []ui.Op {
	var out []ui.Op
	for _, op := range l.Ops {
		if op.Kind == ui.OpImage && op.Asset.Source == screenpack.FromPack && op.Asset.Path == path {
			out = append(out, op)
		}
	}
	return out
}

func near(t *testing.T, what string, got, want float32) {
	t.Helper()
	d := got - want
	if d < -0.01 || d > 0.01 {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// measurer measures text with the test font at a pixel size, independently of
// the code under test.
type measurer struct{ face font.Face }

func newMeasurer(t *testing.T, px float64) measurer {
	t.Helper()
	f, err := opentype.Parse(fontBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	return measurer{face}
}

func (m measurer) width(s string) float32 {
	var w float32
	for _, r := range s {
		adv, _ := m.face.GlyphAdvance(r)
		w += float32(adv) / 64
	}
	return w
}

var (
	red   = ui.Color{R: 1, G: 0, B: 0, A: 1}
	green = ui.Color{R: 0, G: 1, B: 0, A: 1}
)
