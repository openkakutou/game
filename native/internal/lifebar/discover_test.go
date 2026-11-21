package lifebar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverListsValidFoldersAndReportsBrokenOnes(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "zeta", "FIGHT.DEF"), "[Lifebar]\n")
	write(t, filepath.Join(dir, "default", "fight.def"), "[Lifebar]\n")
	write(t, filepath.Join(dir, "broken", "fight.def"), "[Info]\n")
	write(t, filepath.Join(dir, "empty", "readme.txt"), "x")
	write(t, filepath.Join(dir, "loose.def"), "[Lifebar]\n")

	entries, problems := Discover(os.DirFS(dir), ".")
	if len(entries) != 2 || entries[0].Name != "default" || entries[1].Name != "zeta" {
		t.Fatalf("entries = %+v", entries)
	}
	if len(problems) != 2 {
		t.Fatalf("problems = %v", problems)
	}
	if entries, problems := Discover(os.DirFS(dir), "nope"); entries != nil || problems != nil {
		t.Error("a missing lifebars/ folder is simply empty")
	}
}

func TestPick_PrefersTheExactNameOverACaseVariant(t *testing.T) {
	entries := []Entry{{Name: "Alt"}, {Name: "alt"}}
	if e, _ := Pick(entries, "alt"); e.Name != "alt" {
		t.Errorf("pick alt = %q", e.Name)
	}
	if e, _ := Pick(entries, "Alt"); e.Name != "Alt" {
		t.Errorf("pick Alt = %q", e.Name)
	}
	if e, _ := Pick(entries, "ALT"); e.Name != "Alt" {
		t.Errorf("pick ALT = %q, want the first match ignoring case", e.Name)
	}
}

func TestPick(t *testing.T) {
	entries := []Entry{{Name: "default"}, {Name: "vhd"}}
	if e, ok := Pick(entries, "VHD"); !ok || e.Name != "vhd" {
		t.Errorf("pick = %v %v", e, ok)
	}
	if e, ok := Pick(entries, "gone"); !ok || e.Name != "default" {
		t.Errorf("fallback = %v %v", e, ok)
	}
	if _, ok := Pick(entries[1:], "gone"); ok {
		t.Error("no selection and no default: nothing")
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
