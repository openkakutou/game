package host

import (
	"io/fs"
	"os"
	"strings"

	"github.com/openkakutou/game/native/internal/game"
	"path/filepath"
	"testing"
)

func TestContentRoot_IsTheProgramFolder(t *testing.T) {
	exe := filepath.Join("games", "kakutou", "openkakutou")
	if got, want := ContentRoot(exe), filepath.Join("games", "kakutou"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestContentRoot_IsTheFolderHoldingTheAppBundleOnMac(t *testing.T) {
	exe := filepath.Join("Applications", "Fight", "Kakutou.APP", "Contents", "MacOS", "openkakutou")
	if got, want := ContentRoot(exe), filepath.Join("Applications", "Fight"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestContentRoot_IgnoresMacOSFolderOutsideAnAppBundle(t *testing.T) {
	exe := filepath.Join("x", "Contents", "MacOS", "openkakutou")
	if got, want := ContentRoot(exe), filepath.Join("x", "Contents", "MacOS"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMapPaths_MapsAbsolutePathsInsideTheGameFolderToFileNames(t *testing.T) {
	root := t.TempDir()
	cl := game.ParseArgs([]string{filepath.Join(root, "chars", "kfm", "kfm.def"), "rival", "-s", filepath.Join(root, "stages", "Dojo")})
	MapPaths(cl, root)
	if got, want := cl.Flags["-p1"], "chars/kfm/kfm.def"; got != want {
		t.Errorf("-p1 = %q, want %q", got, want)
	}
	if got := cl.Flags["-p2"]; got != "rival" {
		t.Errorf("-p2 = %q, a name must stay as typed", got)
	}
	if got, want := cl.Flags["-s"], "stages/Dojo"; got != want {
		t.Errorf("-s = %q, want %q", got, want)
	}
}

// copyFighter copies the fixture character "fighter" into dir/fighter.
func copyFighter(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "content", "chars", "fighter")
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "fighter"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "fighter", e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "fighter", "fighter.def")
}

func TestOpenAndMapPaths_AnAbsolutePathOutsideTheGameFolderLoadsWithItsSiblingFiles(t *testing.T) {
	root := t.TempDir()
	def := copyFighter(t, t.TempDir())
	cl := game.ParseArgs([]string{def, "rival"})
	MapPaths(cl, root)
	fsys := Open(root)
	name, err := game.ResolveCharDef(fsys, cl.Flags["-p1"])
	if err != nil {
		t.Fatalf("an absolute path outside the game folder must load: %v", err)
	}
	fighter, err := game.LoadFighter(fsys, name)
	if err != nil {
		t.Fatal(err)
	}
	if len(fighter.Cmd.Commands) == 0 || fighter.Char.Name != "fighter" {
		t.Errorf("sibling files not loaded: name %q, %d commands", fighter.Char.Name, len(fighter.Cmd.Commands))
	}
	if data, err := fs.ReadFile(fsys, fighter.SffPath); err != nil || len(data) == 0 {
		t.Errorf("sprite file %q unreadable: %v", fighter.SffPath, err)
	}
}

func TestOpenAndMapPaths_AnAbsolutePathMatchesWithoutCaseAndWithoutDefExtension(t *testing.T) {
	root := t.TempDir()
	parent := t.TempDir()
	def := copyFighter(t, parent)
	cl := game.ParseArgs([]string{filepath.Join(parent, "FIGHTER"), "rival"})
	MapPaths(cl, root)
	got, err := game.ResolveCharDef(Open(root), cl.Flags["-p1"])
	if err != nil {
		t.Fatal(err)
	}
	// The temp dir may sit behind a symlink (macOS: /var -> /private/var),
	// which MapPaths resolves.
	realDef, err := filepath.EvalSymlinks(def)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := absName(realDef)
	if !strings.EqualFold(got, want) || got != want {
		t.Errorf("got %q, want the stored spelling %q", got, want)
	}
}

func TestOpen_AbsNameReflectsAbsolutePathsAndIsReadOnlyToTheCore(t *testing.T) {
	dir := t.TempDir()
	name, ok := absName(filepath.Join(dir, "x.txt"))
	if !ok || !strings.HasPrefix(name, "@abs/") || !strings.HasSuffix(name, "/x.txt") || !fs.ValidPath(name) {
		t.Fatalf("absName = %q, %v", name, ok)
	}
}

func TestOpen_ReadsFilesOfTheGameFolder(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(Open(root), "a.txt")
	if err != nil || string(data) != "hello" {
		t.Fatalf("got %q, %v", data, err)
	}
}

func TestMapPaths_KeepsAPathInsideASymlinkedGameFolderWhenOnlyTheCoreLookupFindsIt(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(filepath.Join(real, "chars", "kfm"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	// "KFM" exists only through the core's case-insensitive lookup, so the
	// whole path cannot be resolved; its symlinked prefix still must be.
	cl := game.ParseArgs([]string{"-p1", filepath.Join(link, "chars", "KFM"), "-p2", "kfm"})
	MapPaths(cl, link)
	if got := cl.Flags["-p1"]; got != "chars/KFM" {
		t.Errorf("-p1 = %q, want the game-folder name %q", got, "chars/KFM")
	}
}
