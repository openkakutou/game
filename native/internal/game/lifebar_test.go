package game

import (
	"github.com/openkakutou/engine/v2/round"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// selectLifebar discovers the lifebars of root and selects name.
func selectLifebar(root, name string) (*HUDLifebar, []string) {
	fsys := os.DirFS(root)
	entries, problems := DiscoverLifebars(fsys)
	hud, notices := SelectLifebar(fsys, entries, name)
	return hud, append(problems, notices...)
}

func TestSelectLifebarUsesTheNamedFolderWhateverItsCase(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "lifebars", "default", "fight.def"), "[Lifebar]\n")
	put(t, filepath.Join(root, "lifebars", "VHD", "FIGHT.DEF"), "[Files]\nsff = FightHD.sff\n[Lifebar]\n")
	put(t, filepath.Join(root, "lifebars", "VHD", "fighthd.sff"), "x")

	hud, notices := selectLifebar(root, "vhd")
	if hud == nil || filepath.Base(filepath.Dir(hud.Def)) != "VHD" {
		t.Fatalf("hud = %+v, notices %v", hud, notices)
	}
	if hud.Sprites != "lifebars/VHD/fighthd.sff" {
		t.Errorf("sprites found case-insensitively: %q", hud.Sprites)
	}
	if len(notices) != 0 {
		t.Errorf("notices = %v", notices)
	}
}

func TestSelectLifebarFallsBackToDefault(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "lifebars", "default", "fight.def"), "[Files]\nsff = gone.sff\n[Lifebar]\n")
	put(t, filepath.Join(root, "lifebars", "broken", "fight.def"), "nothing")

	hud, notices := selectLifebar(root, "missing")
	if hud == nil || filepath.Base(filepath.Dir(hud.Def)) != "default" {
		t.Fatalf("hud = %+v", hud)
	}
	if hud.Sprites != "" {
		t.Errorf("a missing sprite file is dropped, got %q", hud.Sprites)
	}
	all := strings.Join(notices, "\n")
	for _, want := range []string{`"broken" skipped`, `"missing" not found`, `"gone.sff" not found`} {
		if !strings.Contains(all, want) {
			t.Errorf("notices lack %q:\n%s", want, all)
		}
	}
}

func TestSelectLifebarWithoutLifebarsKeepsBuiltInBars(t *testing.T) {
	root := t.TempDir()
	hud, notices := selectLifebar(root, "default")
	if hud != nil || len(notices) != 0 {
		t.Fatalf("hud %v notices %v", hud, notices)
	}
	hud, notices = selectLifebar(root, "gone")
	if hud != nil || len(notices) != 1 || !strings.Contains(notices[0], `lifebar "gone" not found`) {
		t.Fatalf("a missing selection: hud %v notices %v", hud, notices)
	}
}

func TestSelectLifebarLoadsItsFontsAndNamesTheMissingOnes(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "lifebars", "default", "fight.def"), "[Files]\nfont1 = num1.fnt\nfont2 = NUM1.DEF\nfont3 = gone.fnt\n[Lifebar]\n")
	for _, f := range []string{"num1.fnt", "num1.def", "num1.sff"} {
		raw, err := os.ReadFile(filepath.Join(reference, "fonts", f))
		if err != nil {
			t.Fatal(err)
		}
		put(t, filepath.Join(root, "lifebars", "default", "font", f), string(raw))
	}
	hud, notices := selectLifebar(root, "default")
	if hud == nil || hud.Fonts[1] == nil || hud.Fonts[2] == nil {
		t.Fatalf("fonts 1 (v1 in font/) and 2 (v2, other case) must load: %+v, notices %v", hud, notices)
	}
	if hud.Fonts[3] != nil || len(notices) != 1 || !strings.Contains(notices[0], "font3") || !strings.Contains(notices[0], "built-in font") {
		t.Fatalf("notices = %v, want one naming font3 and the built-in fallback", notices)
	}
}

// fs.go:3298-3312: the lifebar's [Round] times run the round clock; unset
// ones are Ikemen GO's defaults.
func TestSetup_RoundTimingsDefaultTheUnsetOnes(t *testing.T) {
	got := Setup{Timings: round.Timings{CtrlTime: 12, OverTime: 99}}.roundTimings()
	want := round.Timings{StartWaitTime: 30, CtrlTime: 12, OverWaitTime: 45, OverWinTime: 45, OverTime: 99}
	if got != want {
		t.Errorf("timings = %+v, want %+v", got, want)
	}
}
