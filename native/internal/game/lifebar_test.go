package game

import (
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
