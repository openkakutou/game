package game

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/openkakutou/stage"
)

func TestDiscoverCharacters_FindsDefWhateverTheCaseOfFolderAndFile(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "KFM", "kfm.DEF"))
	got, err := DiscoverCharacters(os.DirFS(root), ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "KFM" || got[0].DefPath != "KFM/kfm.DEF" {
		t.Fatalf("got %+v", got)
	}
}

func TestDiscoverCharacters_SkipsSymlinkedFolders(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	other := t.TempDir()
	touch(t, filepath.Join(other, "ghost", "ghost.def"))
	if err := os.Symlink(filepath.Join(other, "ghost"), filepath.Join(root, "ghost")); err != nil {
		t.Fatal(err)
	}
	got, err := DiscoverCharacters(os.DirFS(root), ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("symlinked folder listed: %+v", got)
	}
}

func TestDiscoverCharacters_SkipsSymlinkedDef(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	other := t.TempDir()
	touch(t, filepath.Join(other, "real.def"))
	if err := os.MkdirAll(filepath.Join(root, "ghost"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(other, "real.def"), filepath.Join(root, "ghost", "ghost.def")); err != nil {
		t.Fatal(err)
	}
	got, _ := DiscoverCharacters(os.DirFS(root), ".")
	if len(got) != 0 {
		t.Fatalf("symlinked def listed: %+v", got)
	}
}

func TestDiscoverStages_SkipsSymlinkedDefAndFindsUpperCaseExtension(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Stage.DEF"), []byte("[BGdef]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		other := t.TempDir()
		if err := os.WriteFile(filepath.Join(other, "x.def"), []byte("[BGdef]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(other, "x.def"), filepath.Join(root, "link.def")); err != nil {
			t.Fatal(err)
		}
	}
	got, err := DiscoverStages(os.DirFS(root), ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Stage" {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadFighter_ResolvesSpriteAndCommandFilesWhateverTheirCase(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(content, "chars", "fighter")
	dir := filepath.Join(root, "KFM")
	for from, to := range map[string]string{"fighter.sff": "kfm.sff", "fighter.cmd": "kfm.cmd", "fighter.cns": "kfm.cns", "fighter.air": "kfm.air"} {
		data, err := os.ReadFile(filepath.Join(src, from))
		if err != nil {
			t.Fatal(err)
		}
		touch(t, filepath.Join(dir, to))
		if err := os.WriteFile(filepath.Join(dir, to), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	def := "[Info]\nname = \"kfm\"\n[Files]\ncmd = Kfm.CMD\ncns = Kfm.CNS\nsprite = Kfm.SFF\nanim = Kfm.AIR\n"
	if err := os.WriteFile(filepath.Join(dir, "kfm.DEF"), []byte(def), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := LoadFighter(os.DirFS(root), "KFM/kfm.DEF")
	if err != nil {
		t.Fatal(err)
	}
	if want := "KFM/kfm.sff"; f.SffPath != want {
		t.Fatalf("SffPath %q, want %q", f.SffPath, want)
	}
	ref, err := LoadFighter(os.DirFS(content), "chars/fighter/fighter.def")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Cmd.Commands) != len(ref.Cmd.Commands) || len(f.Cmd.Commands) == 0 {
		t.Fatalf("commands not loaded through the differently-cased .cmd: %d vs %d", len(f.Cmd.Commands), len(ref.Cmd.Commands))
	}
}

func TestStageSpritePath_FindsSpriteWhateverItsCase(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "Stage0.SFF"))
	st := &stage.Stage{BGdef: stage.BGdef{SpriteFile: "stage0.sff"}}
	if got, want := StageSpritePath(os.DirFS(dir), "a.def", st), "Stage0.SFF"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMusic_FindsMusicFileWhateverItsCase(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "Theme.OGG"))
	f := toMatch(t)
	f.FS = os.DirFS(dir)
	f.StageDefPath = "a.def"
	f.Stage.MusicFile = "theme.ogg"
	if p, _, _ := f.Music(); p != "Theme.OGG" {
		t.Fatalf("got %q", p)
	}
}

func TestLoadFighter_DoesNotLoadAnEscapingCommandFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "kfm")
	src := filepath.Join(content, "chars", "fighter")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sff", "cns", "air", "cmd"} {
		data, err := os.ReadFile(filepath.Join(src, "fighter."+name))
		if err != nil {
			t.Fatal(err)
		}
		target := dir
		if name == "cmd" {
			target = root
		}
		if err := os.WriteFile(filepath.Join(target, "kfm."+name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	def := "[Info]\nname = \"kfm\"\n[Files]\ncmd = ..\\kfm.cmd\ncns = kfm.cns\nsprite = kfm.sff\nanim = kfm.air\n"
	if err := os.WriteFile(filepath.Join(dir, "kfm.def"), []byte(def), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := LoadFighter(os.DirFS(root), "kfm/kfm.def")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Cmd.Commands) != 0 {
		t.Errorf("escaping .cmd was loaded")
	}
}

func TestStageSpritePath_RejectsSymlinkedEscapingAndAbsoluteRefs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	stages := filepath.Join(root, "stages")
	touch(t, filepath.Join(root, "outside.sff"))
	if err := os.MkdirAll(stages, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "outside.sff"), filepath.Join(stages, "link.sff")); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"link.sff", `..\outside.sff`, filepath.Join(root, "outside.sff")} {
		st := &stage.Stage{BGdef: stage.BGdef{SpriteFile: ref}}
		if got := StageSpritePath(os.DirFS(root), "stages/a.def", st); got != "" {
			t.Errorf("%q offered as %q", ref, got)
		}
	}
}

func TestStageSpritePath_FallsBackToTheContentRoot(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "stages", "Stage0.SFF"))
	st := &stage.Stage{BGdef: stage.BGdef{SpriteFile: "stages/stage0.sff"}}
	got := StageSpritePath(os.DirFS(root), "stages/a.def", st)
	if want := "stages/Stage0.SFF"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStageSpritePath_PrefersTheFileBesideTheDef(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "stages", "x.sff"))
	touch(t, filepath.Join(root, "x.sff"))
	st := &stage.Stage{BGdef: stage.BGdef{SpriteFile: "x.sff"}}
	got := StageSpritePath(os.DirFS(root), "stages/a.def", st)
	if want := "stages/x.sff"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMusic_FallsBackToTheContentRootAndPrefersBesideTheDef(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "sound", "Theme.OGG"))
	f := toMatch(t)
	f.FS = os.DirFS(root)
	f.StageDefPath = "stages/a.def"
	f.Stage.MusicFile = "sound/theme.ogg"
	if p, _, _ := f.Music(); p != "sound/Theme.OGG" {
		t.Fatalf("root fallback: got %q", p)
	}
	touch(t, filepath.Join(root, "stages", "sound", "theme.ogg"))
	f.Stage.MusicFile = "sound/Theme.ogg"
	if p, _, _ := f.Music(); p != "stages/sound/theme.ogg" {
		t.Fatalf("beside def should win: got %q", p)
	}
}

func TestMusic_RejectsAbsoluteSymlinkedAndEscapingRefs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	touch(t, filepath.Join(root, "outside.ogg"))
	if err := os.MkdirAll(filepath.Join(root, "stages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "outside.ogg"), filepath.Join(root, "stages", "l.ogg")); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"l.ogg", `..\outside.ogg`, filepath.Join(root, "outside.ogg")} {
		f := toMatch(t)
		f.FS = os.DirFS(root)
		f.StageDefPath = "stages/a.def"
		f.Stage.MusicFile = ref
		if p, _, _ := f.Music(); p != "" {
			t.Errorf("%q offered as %q", ref, p)
		}
	}
}

func TestLoadFighter_DoesNotLoadASymlinkedCommandFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "kfm")
	src := filepath.Join(content, "chars", "fighter")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sff", "cns", "air"} {
		data, err := os.ReadFile(filepath.Join(src, "fighter."+name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "kfm."+name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(src, "fighter.cmd"), filepath.Join(dir, "kfm.cmd")); err != nil {
		t.Fatal(err)
	}
	def := "[Info]\nname = \"kfm\"\n[Files]\ncmd = kfm.cmd\ncns = kfm.cns\nsprite = kfm.sff\nanim = kfm.air\n"
	if err := os.WriteFile(filepath.Join(dir, "kfm.def"), []byte(def), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := LoadFighter(os.DirFS(root), "kfm/kfm.def")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Cmd.Commands) != 0 {
		t.Errorf("symlinked .cmd was loaded")
	}
}
