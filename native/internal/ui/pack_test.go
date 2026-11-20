package ui_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/ui"
)

func TestUIPackage_ImportsNoFileAccessNorDrawing(t *testing.T) {
	forbidden := []string{"os", "io/ioutil", "github.com/veandco/go-sdl2", "github.com/go-gl"}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
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
			for _, bad := range forbidden {
				if got == bad || strings.HasPrefix(got, bad+"/") {
					t.Errorf("%s imports %q, which the layout must not use", name, got)
				}
			}
		}
	}
}

const redBackground = `"background": [{ "type": "rect", "anchor": "topLeft", "size": [10, 10], "color": "#ff0000" }]`

func TestLoadPack_UsesTheBuiltInScreenpackWhenTheGameFolderHasNone(t *testing.T) {
	pack, notices := ui.LoadPack(fstest.MapFS{})
	if len(notices) != 0 {
		t.Errorf("notices = %v, want none", notices)
	}
	list := draw(ui.NewLayout(pack), menuState([]string{"Versus"}, 0))
	// The built-in title: background.png first, covering 400 x 100 units (2880 x 720 px).
	bg := imageOps(list, "images/background.png")
	if len(bg) != 1 || bg[0].W != 2880 || bg[0].H != 720 {
		t.Errorf("built-in background ops = %+v, want one 2880x720 quad", bg)
	}
}

func TestLoadPack_ScreenpacksDefaultOverridesTheBuiltInOne(t *testing.T) {
	game := gameWith(t, redBackground, nil)
	list := draw(layoutOf(t, game), menuState([]string{"Versus"}, 0))
	if len(rectOps(list, red)) != 1 {
		t.Errorf("the folder's red background is missing: %+v", list.Ops)
	}
	if len(imageOps(list, "images/background.png")) != 0 {
		t.Errorf("the built-in background was drawn over the folder's pack")
	}
}

func TestLoadPack_FindsTheFolderWhateverItsCase(t *testing.T) {
	game := fstest.MapFS{}
	for name, file := range gameWith(t, redBackground, nil) {
		game["ScreenPacks/Default"+strings.TrimPrefix(name, "screenpacks/default")] = file
	}
	list := draw(layoutOf(t, game), menuState([]string{"Versus"}, 0))
	if len(rectOps(list, red)) != 1 {
		t.Errorf("the folder ScreenPacks/Default was not used: %+v", list.Ops)
	}
}

func TestLoadPack_AnInvalidFolderFallsBackToTheBuiltInWithANotice(t *testing.T) {
	game := fstest.MapFS{"screenpacks/default/screenpack.json": {Data: []byte(`{}`)}}
	pack, notices := ui.LoadPack(game)
	if len(notices) != 1 || !strings.Contains(notices[0], "screenpacks/default") {
		t.Fatalf("notices = %q, want one naming screenpacks/default", notices)
	}
	list := draw(ui.NewLayout(pack), menuState([]string{"Versus"}, 0))
	if len(imageOps(list, "images/logo.png")) != 1 {
		t.Errorf("the built-in logo is missing after the fallback: %+v", list.Ops)
	}
}

func TestLoadPack_AFontThatCannotBeParsedIsARefusedFolder(t *testing.T) {
	game := gameWith(t, redBackground, fstest.MapFS{"fonts/NotoSans-Regular.ttf": {Data: []byte("not a font")}})
	_, notices := ui.LoadPack(game)
	if len(notices) != 1 || !strings.Contains(notices[0], "font") {
		t.Errorf("notices = %q, want one about the font", notices)
	}
}

func TestLoadPack_ASymbolicLinkInPlaceOfTheFolderIsRefusedWithANotice(t *testing.T) {
	game := fstest.MapFS{
		"screenpacks/default": {Mode: fs.ModeSymlink, Data: []byte("elsewhere")},
	}
	pack, notices := ui.LoadPack(game)
	if len(notices) != 1 || !strings.Contains(notices[0], "symbolic link") {
		t.Fatalf("notices = %q, want one saying screenpacks/default is a symbolic link", notices)
	}
	list := draw(ui.NewLayout(pack), menuState([]string{"Versus"}, 0))
	if len(imageOps(list, "images/logo.png")) != 1 {
		t.Errorf("the built-in logo is missing after the fallback")
	}
}
