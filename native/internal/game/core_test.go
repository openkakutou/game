package game

import (
	"github.com/openkakutou/game/native/internal/mode"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/engine/v2/input"
)

// forbiddenImports are what the core must never reach: file access of the
// host process, and the screen or sound device.
var forbiddenImports = []string{"os", "io/ioutil", "github.com/veandco/go-sdl2", "github.com/go-gl"}

// The core and the packages it imports (mode, lifebar, bmfont) read files
// only through the fs.FS they are given, and draw nothing.
func TestCore_ImportsNoFileAccessNorDrawing(t *testing.T) {
	var files []string
	for _, dir := range []string{".", "../mode", "../lifebar", "../bmfont"} {
		found, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, found...)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range parsed.Imports {
			got := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbiddenImports {
				if got == bad || strings.HasPrefix(got, bad+"/") {
					t.Errorf("%s imports %q, which the core must not use", name, got)
				}
			}
		}
	}
}

// memoryCopy copies the folder dir into an in-memory file system.
func memoryCopy(t *testing.T, dir string) fstest.MapFS {
	t.Helper()
	mem := fstest.MapFS{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		mem[filepath.ToSlash(rel)] = &fstest.MapFile{Data: data}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return mem
}

// scriptedMatch plays the first two characters on the first stage for ticks
// ticks, P1 tapping "a" and P2 doing nothing, on a close-quarters stage.
func scriptedMatch(t *testing.T, fsys fs.FS, ticks int) *Flow {
	t.Helper()
	chars, err := DiscoverCharacters(fsys, "chars")
	if err != nil {
		t.Fatal(err)
	}
	stages, err := DiscoverStages(fsys, "stages")
	if err != nil {
		t.Fatal(err)
	}
	f := NewFlow(fsys, chars, stages, DefaultLoader(fsys), []mode.Mode{versusMode(t)})
	inMode(t, f)
	f.Navigate(NavConfirm)
	f.Navigate(NavRight)
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm)
	f.SetupRow = SetupRowStart
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm) // VS screen
	if f.Screen != ScreenMatch {
		t.Fatalf("screen=%v msg=%q", f.Screen, f.Message)
	}
	m, err := NewMatch(f.Fighters[0], f.Fighters[1], closeStage(), f.Setup())
	if err != nil {
		t.Fatal(err)
	}
	f.Match = m
	for i := 0; i < ticks; i++ {
		f.Tick([2]input.TickInput{tapA(i), {}})
	}
	return f
}

func TestCore_ScriptedMatchFromMemoryPlaysLikeTheSameMatchFromDisk(t *testing.T) {
	const ticks = 400
	disk := scriptedMatch(t, os.DirFS(content), ticks)
	memory := scriptedMatch(t, memoryCopy(t, content), ticks)

	// The script must have done something, or equal states prove nothing:
	// P1's attacks hit P2 on the close-quarters stage.
	if disk.Match.State.Fighters[1].Health >= DefaultHealth {
		t.Fatalf("scripted match left P2 unhurt (health %d): the comparison is empty", disk.Match.State.Fighters[1].Health)
	}
	if memory.Match.State != disk.Match.State {
		t.Errorf("in-memory run diverged from the disk run:\n memory %+v\n disk   %+v", memory.Match.State, disk.Match.State)
	}
	if memory.Fighters[0].SffPath != "chars/fighter/fighter.sff" {
		t.Errorf("sprite file name = %q, want the fs name chars/fighter/fighter.sff", memory.Fighters[0].SffPath)
	}
}
