package bmfont

import (
	"os"
	"testing"
	"testing/fstest"
)

const fonts = "../../testdata/reference/fonts"

func load(t *testing.T, name string) *Font {
	t.Helper()
	f, err := Load(os.DirFS(fonts), name)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// lit counts the opaque pixels of a glyph.
func lit(g *Glyph) int {
	n := 0
	for _, p := range g.Pixels {
		if p.A != 0 {
			n++
		}
	}
	return n
}

func TestLoad_V1NumberFontWidthsAndSpacing(t *testing.T) {
	f := load(t, "num1.fnt")
	// num1.fnt: Size 9,13, Spacing 1,1, '1' and '0' 9 wide, 'o' 14 wide.
	if got := f.TextWidth("10", 0); got != 19 {
		t.Errorf("width of 10 = %d, want 9+1+9", got)
	}
	if got := f.TextWidth("o", 0); got != 14 {
		t.Errorf("width of o = %d, want 14", got)
	}
	if got := f.TextWidth("", 0); got != 0 {
		t.Errorf("width of empty text = %d, want 0", got)
	}
}

func TestLoad_V1GlyphIsCutFromTheStripWithTransparentIndexZero(t *testing.T) {
	f := load(t, "num1.fnt")
	g := f.Glyph(0, '1')
	if g == nil || g.W != 9 || g.H != 13 || len(g.Pixels) != 9*13 {
		t.Fatalf("glyph 1 = %+v, want 9x13", g)
	}
	if n := lit(g); n == 0 || n == len(g.Pixels) {
		t.Fatalf("glyph 1 has %d opaque pixels of %d: want a shape, not empty nor a full block", n, len(g.Pixels))
	}
}

func TestLoad_V1BanksRecolorTheSameShape(t *testing.T) {
	f := load(t, "num1.fnt")
	a, b := f.Glyph(0, '8'), f.Glyph(1, '8')
	if a == nil || b == nil || lit(a) != lit(b) {
		t.Fatalf("banks 0 and 1 must draw the same shape")
	}
	same := true
	for i := range a.Pixels {
		if a.Pixels[i] != b.Pixels[i] {
			same = false
		}
	}
	if same {
		t.Fatal("bank 1 has the colors of bank 0: palette banks are not applied")
	}
	if f.Glyph(99, '8') == nil {
		t.Fatal("an out-of-range bank falls back to bank 0, as Ikemen GO")
	}
}

func TestLoad_V2PaletteFont(t *testing.T) {
	f := load(t, "num1.def")
	if got := f.TextWidth("10", 0); got != 19 {
		t.Errorf("width of 10 = %d, want 19", got)
	}
	a, b := f.Glyph(0, '8'), f.Glyph(1, '8')
	if a == nil || b == nil || a.W != 9 || lit(a) == 0 || lit(a) != lit(b) {
		t.Fatalf("glyph 8 banks 0/1 = %+v / %+v", a, b)
	}
	if a.Pixels[firstLit(a)] == b.Pixels[firstLit(a)] {
		t.Fatal("v2 bank 1 must use the sff's palette 0,1")
	}
}

func firstLit(g *Glyph) int {
	for i, p := range g.Pixels {
		if p.A != 0 {
			return i
		}
	}
	return 0
}

func TestLayout_AlignmentAndOffsetFollowIkemenGO(t *testing.T) {
	f := load(t, "num1.fnt")
	left := f.Layout("10", 0, 1, 100, 50, 1, nil)
	if len(left) != 2 || left[0].X != 100 || left[1].X != 110 {
		t.Fatalf("left aligned = %+v, want x 100 then 110", left)
	}
	// DrawText: y += offset.y - size.y + 1, so a 13 high font starts 12 above.
	if left[0].Y != 38 {
		t.Errorf("top = %v, want 50-13+1", left[0].Y)
	}
	center := f.Layout("10", 0, 0, 100, 50, 1, nil)
	if center[0].X != 90.5 {
		t.Errorf("centered start = %v, want 100-19/2", center[0].X)
	}
	right := f.Layout("10", 0, -1, 100, 50, 2, nil)
	if right[0].X != 62 || right[1].X != 82 {
		t.Errorf("right aligned at scale 2 = %+v, want 100-38 then +20", right)
	}
}

func TestLayout_UnknownCharacterIsASpace(t *testing.T) {
	f := load(t, "num1.fnt")
	got := f.Layout("1Z1", 0, 1, 0, 0, 1, nil)
	if len(got) != 2 || got[1].X != 9+1+9+1 {
		t.Fatalf("1Z1 = %+v, want Z drawn as a 9 wide space", got)
	}
}

func TestLoad_MissingOrBrokenFontIsAnError(t *testing.T) {
	if _, err := Load(os.DirFS(fonts), "nope.fnt"); err == nil {
		t.Error("missing font: want an error")
	}
	bad := fstest.MapFS{"bad.fnt": {Data: []byte("not a font")}}
	if _, err := Load(bad, "bad.fnt"); err == nil {
		t.Error("broken v1 font: want an error")
	}
	noSff := fstest.MapFS{"x.def": {Data: []byte("[Def]\nType = bitmap\nSize = 8,8\nFile = gone.sff\n")}}
	if _, err := Load(noSff, "x.def"); err == nil {
		t.Error("v2 font whose sff is missing: want an error")
	}
}
