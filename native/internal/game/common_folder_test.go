package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shippedCommon is the game's own common/ folder, as released.
const shippedCommon = "../../../common"

// contentWithShippedCommon builds a content folder holding a reference
// character under chars/ and a copy of the shipped common/ folder.
func contentWithShippedCommon(t *testing.T, char string) string {
	t.Helper()
	root := t.TempDir()
	copyDir(t, filepath.Join(reference, "chars", char), filepath.Join(root, "chars", char))
	copyDir(t, shippedCommon, filepath.Join(root, "common"))
	return root
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestShippedCommon_GivesKungFuManItsCommonStates(t *testing.T) {
	fsys := os.DirFS(contentWithShippedCommon(t, "kfm"))
	f, err := LoadFighter(fsys, "chars/kfm/kfm.def", CommonFolder(fsys))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Missing) != 0 {
		t.Fatalf("stcommon = common1.cns must resolve to the shipped common1.cns.zss, missing %v", f.Missing)
	}
	m := startMatch(t, f, f)
	requireStartsTheRound(t, m)
}

func TestShippedCommon_ReplacingCommon1ChangesTheStates(t *testing.T) {
	root := contentWithShippedCommon(t, "kfm")
	if err := os.Remove(filepath.Join(root, "common", "common1.cns.zss")); err != nil {
		t.Fatal(err)
	}
	fsys := os.DirFS(root)
	f, err := LoadFighter(fsys, "chars/kfm/kfm.def", CommonFolder(fsys))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Missing) != 1 || f.Missing[0] != "common1.cns" {
		t.Fatalf("without the shipped file, Missing = %v, want [common1.cns]", f.Missing)
	}
}

func TestShippedCommon_EveryFileHasItsLicenseListed(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(shippedCommon, "LICENSES.md"))
	if err != nil {
		t.Fatal(err)
	}
	listed := string(raw)
	entries, err := os.ReadDir(shippedCommon)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == "LICENSES.md" {
			continue
		}
		if !strings.Contains(listed, "| `"+e.Name()+"` |") {
			t.Errorf("common/%s ships without a row in common/LICENSES.md", e.Name())
		}
	}
}
