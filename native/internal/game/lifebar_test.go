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

func TestSelectLifebarUsesTheAppearanceChoice(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "lifebars", "default", "fight.def"), "[Lifebar]\n")
	put(t, filepath.Join(root, "lifebars", "VHD", "FIGHT.DEF"), "[Files]\nsff = FightHD.sff\n[Lifebar]\n")
	put(t, filepath.Join(root, "lifebars", "VHD", "fighthd.sff"), "x")
	put(t, filepath.Join(root, "config", "appearance.json"), `{"lifebar":"vhd"}`)

	hud, notices := SelectLifebar(os.DirFS(root))
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
	put(t, filepath.Join(root, "config", "appearance.json"), `{"lifebar":"missing"}`)

	hud, notices := SelectLifebar(os.DirFS(root))
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
	hud, notices := SelectLifebar(os.DirFS(root))
	if hud != nil || len(notices) != 0 {
		t.Fatalf("hud %v notices %v", hud, notices)
	}
	put(t, filepath.Join(root, "config", "appearance.json"), `{"lifebar":`)
	put(t, filepath.Join(root, "lifebars", "x", "fight.def"), "[Lifebar]\n")
	hud, notices = SelectLifebar(os.DirFS(root))
	if hud != nil || len(notices) != 1 || !strings.Contains(notices[0], "appearance settings ignored") {
		t.Fatalf("corrupt settings: hud %v notices %v", hud, notices)
	}
}
