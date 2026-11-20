package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"
)

// appearanceFlow is a flow on Options whose game folder has the lifebars
// Alt and default, and whose installed screenpacks are alt, default and zeta.
func appearanceFlow(t *testing.T, files map[string]string) (*Flow, *recorder) {
	t.Helper()
	root := t.TempDir()
	put(t, filepath.Join(root, "lifebars", "default", "fight.def"), "[Lifebar]\n")
	put(t, filepath.Join(root, "lifebars", "Alt", "fight.def"), "[Lifebar]\n")
	fsys := os.DirFS(root)
	rec := &recorder{}
	f := newTitleFlow(t)
	f.FS = fsys
	f.Settings = LoadSettings(configFS(files), "linux", rec.write, nil)
	f.Packs = []string{"alt", "default", "zeta"}
	f.Lifebars, _ = DiscoverLifebars(fsys)
	f.Lifebar, _ = SelectLifebar(fsys, f.Lifebars, f.Settings.Appearance.Lifebar)
	rec.names = nil
	navAll(f, NavConfirm, NavDown, NavConfirm)                // main menu, Options
	navAll(f, NavDown, NavDown, NavDown, NavDown, NavConfirm) // Appearance
	if f.Screen != ScreenOptionsAppearance {
		t.Fatalf("screen = %v, want the appearance options", f.Screen)
	}
	return f, rec
}

func TestOptions_AppearanceListsScreenpackAndLifebarThenBackAndLeftRightCycleTheInstalledNames(t *testing.T) {
	f, _ := appearanceFlow(t, nil)
	if want := []string{"Screenpack  default", "Lifebar  default", "Back"}; !sameStrings(f.OptionsItems(), want) {
		t.Fatalf("rows = %q, want %q", f.OptionsItems(), want)
	}
	if f.OptionsTitle() != "Appearance" {
		t.Errorf("title = %q", f.OptionsTitle())
	}
	f.Navigate(NavRight)
	if f.OptionsItems()[0] != "Screenpack  zeta" {
		t.Errorf("right: %q, want zeta", f.OptionsItems())
	}
	f.Navigate(NavRight)
	if f.OptionsItems()[0] != "Screenpack  alt" {
		t.Errorf("right again wraps: %q, want alt", f.OptionsItems())
	}
	f.Navigate(NavLeft)
	if f.OptionsItems()[0] != "Screenpack  zeta" {
		t.Errorf("left wraps back: %q, want zeta", f.OptionsItems())
	}
	navAll(f, NavDown, NavRight) // Lifebar: Alt, default
	if f.OptionsItems()[1] != "Lifebar  Alt" {
		t.Errorf("lifebar right wraps: %q, want Alt", f.OptionsItems())
	}
}

func TestOptions_AppearanceIsPreviewedWhileBrowsingAndAppliedAndSavedOnceWhenLeft(t *testing.T) {
	f, rec := appearanceFlow(t, nil)
	if f.PreviewPack() != "default" {
		t.Errorf("preview on entry = %q, want default", f.PreviewPack())
	}
	gen := f.Settings.AppearanceGen
	f.Navigate(NavRight)
	if f.PreviewPack() != "zeta" {
		t.Errorf("preview = %q, want the candidate zeta", f.PreviewPack())
	}
	if f.Settings.AppearanceGen != gen || len(rec.names) != 0 {
		t.Fatalf("gen %d (was %d), written %q: nothing applies or saves before leaving the page", f.Settings.AppearanceGen, gen, rec.names)
	}
	navAll(f, NavDown, NavDown, NavConfirm) // Back
	if f.Screen != ScreenOptions || f.OptionsCursor() != int(ConcernAppearance) {
		t.Errorf("screen = %v cursor = %d, want Options on Appearance", f.Screen, f.OptionsCursor())
	}
	if f.PreviewPack() != "" {
		t.Errorf("preview after leaving = %q, want none", f.PreviewPack())
	}
	if f.Settings.AppearanceGen != gen+1 || f.ScreenpackChoice() != "zeta" {
		t.Errorf("gen = %d choice = %q, want %d and zeta", f.Settings.AppearanceGen, f.ScreenpackChoice(), gen+1)
	}
	if !slices.Equal(rec.names, []string{"config/appearance.json"}) {
		t.Fatalf("written = %q, want exactly config/appearance.json", rec.names)
	}
	got := decodeJSON(t, rec.files["config/appearance.json"])
	if got["screenpack"] != "zeta" || got["lifebar"] != "default" {
		t.Errorf("appearance.json = %v", got)
	}
}

func TestOptions_LeavingAppearanceUnchangedOrBackOnTheOldChoiceSavesAndAppliesNothing(t *testing.T) {
	f, rec := appearanceFlow(t, nil)
	gen := f.Settings.AppearanceGen
	navAll(f, NavRight, NavLeft, NavCancel) // zeta, back to default, leave with cancel
	if f.Screen != ScreenOptions || len(rec.names) != 0 || f.Settings.AppearanceGen != gen {
		t.Errorf("screen %v written %q gen %d (was %d), want nothing saved nor applied", f.Screen, rec.names, f.Settings.AppearanceGen, gen)
	}
}

func TestOptions_LeavingAppearanceWithCancelKeepsTheChoiceToo(t *testing.T) {
	f, rec := appearanceFlow(t, nil)
	navAll(f, NavLeft, NavCancel) // alt
	if f.ScreenpackChoice() != "alt" || len(rec.names) != 1 {
		t.Errorf("choice = %q written = %q, want alt saved", f.ScreenpackChoice(), rec.names)
	}
}

func TestOptions_TheLifebarCandidateIsSampledOnItsRowAndTheHUDUsesItAfterLeaving(t *testing.T) {
	f, _ := appearanceFlow(t, nil)
	if f.PreviewLifebar() != nil {
		t.Error("the sample lifebar is shown only while the Lifebar row is highlighted")
	}
	navAll(f, NavDown, NavRight) // Lifebar row, Alt
	sample := f.PreviewLifebar()
	if sample == nil || sample.Def != "lifebars/Alt/fight.def" {
		t.Fatalf("sample = %+v, want lifebars/Alt/fight.def", sample)
	}
	if f.Lifebar == nil || f.Lifebar.Def != "lifebars/default/fight.def" {
		t.Fatalf("the HUD lifebar changed before leaving: %+v", f.Lifebar)
	}
	navAll(f, NavDown, NavConfirm) // Back
	if f.Lifebar == nil || f.Lifebar.Def != "lifebars/Alt/fight.def" {
		t.Errorf("HUD lifebar = %+v, want lifebars/Alt/fight.def", f.Lifebar)
	}
	if f.Settings.Appearance.Lifebar != "Alt" {
		t.Errorf("saved lifebar = %q, want Alt", f.Settings.Appearance.Lifebar)
	}
}

func TestOptions_ASavedNameThatIsNoLongerInstalledShowsDefaultAndIsKeptUntilChanged(t *testing.T) {
	f, rec := appearanceFlow(t, map[string]string{"appearance.json": `{"screenpack": "gone", "lifebar": "gone"}`})
	if want := []string{"Screenpack  default", "Lifebar  default", "Back"}; !sameStrings(f.OptionsItems(), want) {
		t.Fatalf("rows = %q, want %q", f.OptionsItems(), want)
	}
	f.Navigate(NavCancel)
	if len(rec.names) != 0 || f.Settings.Appearance.Screenpack != "gone" {
		t.Errorf("written %q appearance %+v, want the file untouched", rec.names, f.Settings.Appearance)
	}
	f, rec = appearanceFlow(t, map[string]string{"appearance.json": `{"screenpack": "gone"}`})
	f.Navigate(NavRight) // from default to zeta
	f.Navigate(NavCancel)
	if f.Settings.Appearance.Screenpack != "zeta" || len(rec.names) != 1 {
		t.Errorf("appearance %+v written %q, want zeta saved", f.Settings.Appearance, rec.names)
	}
}

func TestOptions_SaveOpenConcernSavesAChangedAppearanceOnce(t *testing.T) {
	f, rec := appearanceFlow(t, nil)
	f.Navigate(NavRight)
	f.SaveOpenConcern()
	f.SaveOpenConcern()
	if len(rec.names) != 1 || rec.names[0] != "config/appearance.json" {
		t.Errorf("written = %q, want config/appearance.json once", rec.names)
	}
}

func TestSortRosters_OrdersBothListsAndKeepsThePicksOnTheirEntries(t *testing.T) {
	f := NewFlow(fstest.MapFS{}, []Entry{
		{Name: "a", Display: "Alpha"}, {Name: "b", Display: "Bravo"}, {Name: "c", Display: "Charlie"},
	}, []Entry{
		{Name: "x.def", Display: "Zed"}, {Name: "y.def", Display: "Arena"},
	}, Loader{}, nil)
	f.Picks, f.StageNo = [2]int{0, 2}, 0
	f.SortRosters("displayName", true, "displayName", false)
	if f.Chars[0].Name != "c" || f.Chars[1].Name != "b" || f.Chars[2].Name != "a" {
		t.Errorf("chars = %+v, want c, b, a", f.Chars)
	}
	if f.Stages[0].Name != "y.def" || f.Stages[1].Name != "x.def" {
		t.Errorf("stages = %+v, want y.def, x.def", f.Stages)
	}
	if f.Picks != [2]int{2, 0} || f.StageNo != 1 {
		t.Errorf("picks = %v stage = %d, want [2 0] and 1", f.Picks, f.StageNo)
	}
}
