package game

import (
	"image/color"
	"os"
	"path"
	"regexp"
	"testing"
	"testing/fstest"
)

// kfmWithAct is the reference kfm with "pal1 = kfm1.act" in its [Files],
// whose palette index 5 is (1, 2, 3).
func kfmWithAct(t *testing.T, act []byte) fstest.MapFS {
	t.Helper()
	fsys := fstest.MapFS{}
	dir := reference + "/chars/kfm"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		raw, err := os.ReadFile(path.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		fsys["chars/kfm/"+e.Name()] = &fstest.MapFile{Data: raw}
	}
	def := string(fsys["chars/kfm/kfm.def"].Data)
	def = regexp.MustCompile(`(?i)\[Files\]`).ReplaceAllString(def, "[Files]\npal1 = kfm1.act")
	fsys["chars/kfm/kfm.def"] = &fstest.MapFile{Data: []byte(def)}
	if act != nil {
		fsys["chars/kfm/kfm1.act"] = &fstest.MapFile{Data: act}
	}
	return fsys
}

// char.go:4743-4800 (Ikemen GO 54d4148a): a character's first .act is its
// default palette.
func TestLoadFighter_ReadsTheFirstActPalette(t *testing.T) {
	act := make([]byte, 768)
	act[(255-5)*3], act[(255-5)*3+1], act[(255-5)*3+2] = 1, 2, 3 // .act triplets are stored in reverse
	f, err := LoadFighter(kfmWithAct(t, act), "chars/kfm/kfm.def")
	if err != nil {
		t.Fatal(err)
	}
	if f.Palette == nil || f.Palette[5] != (color.RGBA{R: 1, G: 2, B: 3, A: 255}) {
		t.Fatalf("palette = %v, want index 5 = 1,2,3", f.Palette)
	}
}

func TestLoadFighter_AMissingOrBrokenActLeavesTheSpritesPalettes(t *testing.T) {
	for name, act := range map[string][]byte{"missing": nil, "too short": make([]byte, 10)} {
		f, err := LoadFighter(kfmWithAct(t, act), "chars/kfm/kfm.def")
		if err != nil {
			t.Fatal(err)
		}
		if f.Palette != nil {
			t.Errorf("%s act: palette %v, want none", name, f.Palette)
		}
	}
}
