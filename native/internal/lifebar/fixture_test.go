package lifebar

import (
	"bytes"
	shared "github.com/openkakutou/lifebar"
	"os"
	"path/filepath"
	"testing"

	"github.com/openkakutou/sff"
)

// fixtureDir is the original lifebar the smoke test and the fixture tests
// use, drawn here so it carries no third-party art.
const fixtureDir = "../../testdata/content/lifebars/default"

// TestGenerateFixture rewrites the fixture's fight.sff when GEN_LIFEBAR=1:
// solid bars, colour index 0 transparent.
func TestGenerateFixture(t *testing.T) {
	if os.Getenv("GEN_LIFEBAR") != "1" {
		t.Skip("set GEN_LIFEBAR=1 to regenerate the lifebar fixture")
	}
	rect := func(w, h int, border, fill byte) []byte {
		px := make([]byte, w*h)
		for y := range h {
			for x := range w {
				px[y*w+x] = fill
				if x == 0 || y == 0 || x == w-1 || y == h-1 {
					px[y*w+x] = border
				}
			}
		}
		return px
	}
	type spec struct {
		group, w, h  int
		border, fill byte
	}
	var sprites []sff.V2WriteSprite
	for _, s := range []spec{{1, 132, 12, 1, 2}, {2, 128, 8, 4, 4}, {3, 128, 8, 3, 3}, {4, 100, 4, 5, 5}, {5, 104, 8, 1, 2}} {
		sprites = append(sprites, sff.V2WriteSprite{Group: s.group, Width: s.w, Height: s.h,
			Format: sff.V2FormatRaw, ColorDepth: 8, PixelData: rect(s.w, s.h, s.border, s.fill)})
	}
	colors := make([]byte, 256*4)
	for i, c := range [][4]byte{{0, 0, 0, 0}, {10, 10, 16, 255}, {50, 54, 70, 255}, {200, 40, 40, 255}, {250, 210, 60, 255}, {70, 140, 240, 255}} {
		copy(colors[i*4:], c[:])
	}
	palettes := []sff.V2WritePalette{{Group: 1, Number: 1, ColorCount: 256, ColorData: colors}}
	var buf bytes.Buffer
	if err := sff.SerializeV2(&buf, [4]byte{0, 1, 0, 2}, sprites, palettes); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "fight.sff"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFixtureParsesAndItsSpritesExist(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(fixtureDir, "fight.def"))
	if err != nil {
		t.Fatal(err)
	}
	lb, warnings := shared.Parse(string(src))
	if len(warnings) != 0 || !hasLifebarSection(string(src)) {
		t.Fatalf("warnings %v", warnings)
	}
	f, err := os.Open(filepath.Join(fixtureDir, lb.Files.Sprites))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, b := range []shared.Bar{lb.Life[0].Bar, lb.Life[1].Bar, lb.Power[0].Bar, lb.Power[1].Bar} {
		for _, e := range []shared.Element{b.Bg0, b.Mid, b.Front} {
			if !e.HasSprite {
				continue
			}
			if _, _, _, err := sff.ResolveSpritePixels(f, e.Sprite.Group, e.Sprite.Image, nil); err != nil {
				t.Errorf("sprite %v: %v", e.Sprite, err)
			}
		}
	}
}
