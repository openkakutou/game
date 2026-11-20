package ui_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/ui"
)

// addPack puts a valid screenpack in the game folder as screenpacks/<folder>:
// the test root file, the font, and the screens (name -> parts) it lists.
func addPack(t *testing.T, g fstest.MapFS, folder string, screens map[string]string) {
	t.Helper()
	g["screenpacks/"+folder+"/screenpack.json"] = &fstest.MapFile{Data: []byte(rootJSON)}
	g["screenpacks/"+folder+"/fonts/NotoSans-Regular.ttf"] = &fstest.MapFile{Data: fontBytes(t)}
	g["screenpacks/"+folder+"/s.ogg"] = &fstest.MapFile{Data: []byte("sound")}
	for name, parts := range screens {
		g["screenpacks/"+folder+"/screens/"+name+".json"] = &fstest.MapFile{Data: []byte(`{"formatVersion": 1, ` + parts + `}`)}
	}
}

func TestDiscoverPacks_ListsValidFoldersSortedIgnoringCaseAndSkipsTheInvalidOne(t *testing.T) {
	g := fstest.MapFS{"screenpacks/broken/screenpack.json": {Data: []byte(`{}`)}}
	for _, folder := range []string{"Zeta", "alpha", "default", "Beta"} {
		addPack(t, g, folder, nil)
	}
	names, problems := ui.DiscoverPacks(g)
	if got := strings.Join(names, ","); got != "alpha,Beta,default,Zeta" {
		t.Errorf("names = %s, want alpha,Beta,default,Zeta", got)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "screenpacks/broken") {
		t.Errorf("problems = %q, want one naming screenpacks/broken", problems)
	}
}

func TestDiscoverPacks_FindsTheFolderWhateverItsCase(t *testing.T) {
	g := fstest.MapFS{}
	addPack(t, g, "x", nil)
	g["ScreenPacks/Alt/screenpack.json"] = g["screenpacks/x/screenpack.json"]
	g["ScreenPacks/Alt/fonts/NotoSans-Regular.ttf"] = g["screenpacks/x/fonts/NotoSans-Regular.ttf"]
	g["ScreenPacks/Alt/s.ogg"] = g["screenpacks/x/s.ogg"]
	delete(g, "screenpacks/x/screenpack.json")
	delete(g, "screenpacks/x/fonts/NotoSans-Regular.ttf")
	delete(g, "screenpacks/x/s.ogg")
	names, _ := ui.DiscoverPacks(g)
	if got := strings.Join(names, ","); got != "Alt,default" {
		t.Errorf("names = %s, want Alt,default", got)
	}
}

func TestDiscoverPacks_ListsTheBuiltInPackAsDefaultWhenTheFolderIsAbsent(t *testing.T) {
	names, problems := ui.DiscoverPacks(fstest.MapFS{})
	if len(names) != 1 || names[0] != "default" || len(problems) != 0 {
		t.Errorf("names = %q problems = %q, want [default] and none", names, problems)
	}
}

func TestDiscoverPacks_ListsTheBuiltInPackAsDefaultWhenTheFolderIsInvalid(t *testing.T) {
	g := fstest.MapFS{"screenpacks/default/screenpack.json": {Data: []byte(`{}`)}}
	addPack(t, g, "alt", nil)
	names, problems := ui.DiscoverPacks(g)
	if got := strings.Join(names, ","); got != "alt,default" {
		t.Errorf("names = %s, want alt,default", got)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "screenpacks/default") {
		t.Errorf("problems = %q, want one naming screenpacks/default", problems)
	}
}

func TestLoadPack_TheSelectedFolderIsLoadedWhateverTheCaseOfTheName(t *testing.T) {
	g := fstest.MapFS{}
	addPack(t, g, "alt", map[string]string{"title": redBackground})
	pack, notices := ui.LoadPack(g, "ALT")
	if len(notices) != 0 || pack.Name() != "alt" {
		t.Fatalf("name = %q notices = %q, want alt and none", pack.Name(), notices)
	}
	list := draw(ui.NewLayout(pack), menuState([]string{"Versus"}, 0))
	if len(rectOps(list, red)) != 1 {
		t.Errorf("the folder alt's red background is missing: %+v", list.Ops)
	}
}

func TestLoadPack_AMissingSelectionFallsBackToTheDefaultFolderWithANotice(t *testing.T) {
	g := fstest.MapFS{}
	addPack(t, g, "default", map[string]string{"title": redBackground})
	pack, notices := ui.LoadPack(g, "gone")
	if pack.Name() != "default" {
		t.Errorf("name = %q, want default", pack.Name())
	}
	if len(notices) != 1 || !strings.Contains(notices[0], `screenpack "gone" not found`) {
		t.Errorf("notices = %q, want one saying screenpack \"gone\" is not found", notices)
	}
	if len(rectOps(draw(ui.NewLayout(pack), menuState([]string{"Versus"}, 0)), red)) != 1 {
		t.Error("the default folder was not used")
	}
}

func TestLoadPack_AnInvalidSelectionFallsBackToTheBuiltInWithANoticeForEachProblem(t *testing.T) {
	g := fstest.MapFS{
		"screenpacks/bad/screenpack.json":     {Data: []byte(`{}`)},
		"screenpacks/default/screenpack.json": {Data: []byte(`{}`)},
	}
	pack, notices := ui.LoadPack(g, "bad")
	if pack.Name() != "default" || len(notices) != 2 {
		t.Fatalf("name = %q notices = %q, want default and 2 notices", pack.Name(), notices)
	}
	if !strings.Contains(notices[0], "screenpacks/bad") || !strings.Contains(notices[1], "screenpacks/default") {
		t.Errorf("notices = %q", notices)
	}
}

func TestLoadPack_ANameThatLooksLikeAPathIsNotAFolder(t *testing.T) {
	g := fstest.MapFS{}
	addPack(t, g, "default", nil)
	addPack(t, g, "x", nil)
	for name, file := range g {
		if strings.HasPrefix(name, "screenpacks/x/") {
			g["other/"+strings.TrimPrefix(name, "screenpacks/x/")] = file // a valid pack outside screenpacks/
		}
	}
	pack, notices := ui.LoadPack(g, "../other")
	if pack.Name() != "default" || len(notices) != 1 {
		t.Errorf("name = %q notices = %q, want default and one notice", pack.Name(), notices)
	}
}

func TestArrange_SortsTheRostersByThePacksRulesAndKeepsThePicksOnTheirEntries(t *testing.T) {
	g := fstest.MapFS{}
	addPack(t, g, "desc", map[string]string{
		"characterSelect": `"sort": { "by": "displayName", "descending": true }`,
	})
	pack, notices := ui.LoadPack(g, "desc")
	if len(notices) != 0 {
		t.Fatalf("notices = %q", notices)
	}
	f := game.NewFlow(nil, []game.Entry{
		{Name: "a", Display: "Alpha"}, {Name: "b", Display: "Bravo"}, {Name: "c", Display: "Charlie"},
	}, []game.Entry{
		{Name: "x.def", Display: "Zed"}, {Name: "y.def", Display: "Arena"},
	}, game.Loader{}, nil)
	f.Picks = [2]int{0, 2} // Alpha and Charlie
	f.StageNo = 0          // Zed
	pack.Arrange(f)
	var chars []string
	for _, e := range f.Chars {
		chars = append(chars, e.Display)
	}
	if got := strings.Join(chars, ","); got != "Charlie,Bravo,Alpha" {
		t.Errorf("characters = %s, want Charlie,Bravo,Alpha (displayName descending)", got)
	}
	if f.Picks != [2]int{2, 0} {
		t.Errorf("picks = %v, want [2 0]: still Alpha and Charlie", f.Picks)
	}
	// The stage screen has no rule in this pack: the default pack's, displayName ascending.
	if f.Stages[0].Display != "Arena" || f.StageNo != 1 {
		t.Errorf("stages = %+v StageNo = %d, want Arena first and StageNo 1 (still Zed)", f.Stages, f.StageNo)
	}
}
