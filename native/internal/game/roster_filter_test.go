package game

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/openkakutou/game/native/internal/mode"
)

// filterChars and filterStages are the installed content of the roster
// filter tests, in the default (folder name) order.
var (
	filterChars  = []Entry{{Name: "a"}, {Name: "boss"}, {Name: "c"}, {Name: "d"}, {Name: "e"}}
	filterStages = []Entry{{Name: "s0"}, {Name: "s1"}, {Name: "s2"}}
)

// arcadeSrc is a mode called Arcade with the given roster rule ("" for none).
func arcadeSrc(roster string) string {
	if roster != "" {
		roster = `"roster": ` + roster + `,`
	}
	return fmt.Sprintf(`{"formatVersion": 1, "id": "arcade", "name": "Arcade",
	"flow": [%s],
	"rules": {"sides": [{"control": "human", "characters": 1}, {"control": "human", "characters": 1}],
	%s "rounds": 1, "timeLimitSeconds": 0, "opponents": {"kind": "chosen"}, "onWin": "results", "onLose": "results"}}`,
		steps(stepSelect, stepStage, stepMatch), roster)
}

func parseArcade(t *testing.T, roster string) mode.Mode {
	t.Helper()
	m, err := mode.Parse("arcade.jsonc", []byte(arcadeSrc(roster)))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// rosterFlow enters the Arcade mode over filterChars and filterStages.
func rosterFlow(t *testing.T, roster string) *Flow {
	t.Helper()
	failing := Loader{Fighter: func(string) (*Fighter, error) { return nil, errors.New("no loading in this test") }}
	f := NewFlow(nil, slices.Clone(filterChars), slices.Clone(filterStages), failing, []mode.Mode{parseArcade(t, roster)})
	return inMode(t, f)
}

func TestRosterFilter_VisibleCharacters_FollowIncludeAndExcludeIgnoringCase(t *testing.T) {
	cases := []struct {
		name, roster string
		want         []int
	}{
		{"no filter shows everything", ``, []int{0, 1, 2, 3, 4}},
		{"exclude removes", `{"characters": {"select": {"exclude": ["boss"]}}}`, []int{0, 2, 3, 4}},
		{"exclude ignores case", `{"characters": {"select": {"exclude": ["BoSs", "A"]}}}`, []int{2, 3, 4}},
		{"include keeps only the listed", `{"characters": {"select": {"include": ["E", "a"]}}}`, []int{0, 4}},
		{"exclude removes from include", `{"characters": {"select": {"include": ["a", "boss", "c"], "exclude": ["boss"]}}}`, []int{0, 2}},
		{"opponents filter leaves the select alone", `{"characters": {"opponents": {"include": ["boss"]}}}`, []int{0, 1, 2, 3, 4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := rosterFlow(t, c.roster)
			if got := f.VisibleChars(); !slices.Equal(got, c.want) {
				t.Errorf("VisibleChars = %v, want %v", got, c.want)
			}
		})
	}
}

func TestRosterFilter_VisibleStages_FollowIncludeAndExclude(t *testing.T) {
	f := rosterFlow(t, `{"stages": {"select": {"include": ["S0", "s1", "s2"], "exclude": ["S1"]}}}`)
	if got := f.VisibleStages(); !slices.Equal(got, []int{0, 2}) {
		t.Errorf("VisibleStages = %v, want [0 2]", got)
	}
}

func TestRosterFilter_CharacterGrid_MovesOverVisibleEntriesOnly(t *testing.T) {
	f := rosterFlow(t, `{"characters": {"select": {"exclude": ["boss"]}}}`) // cells: a c d e
	f.GridColumns = 2                                                       // a c / d e
	f.Navigate(NavRight)
	if f.Cursor != 1 {
		t.Fatalf("right from the first cell = %d, want 1", f.Cursor)
	}
	f.Navigate(NavRight) // wraps inside the row
	if f.Cursor != 0 {
		t.Fatalf("right from the row's end = %d, want the row's start 0", f.Cursor)
	}
	f.Navigate(NavUp) // wraps over the two rows
	if f.Cursor != 2 {
		t.Fatalf("up from the first row = %d, want the second row's cell 2", f.Cursor)
	}
	f.Navigate(NavLeft)
	if f.Cursor != 3 {
		t.Fatalf("left from cell 2 = %d, want its row's last cell 3", f.Cursor)
	}
}

func TestRosterFilter_CharacterPicks_AreTheRosterIndexesOfTheCellsChosen(t *testing.T) {
	f := rosterFlow(t, `{"characters": {"select": {"exclude": ["boss"]}}}`) // cells: a c d e = roster 0 2 3 4
	f.GridColumns = 0
	f.Navigate(NavRight)
	f.Navigate(NavConfirm) // P1: cell 1 = "c", roster index 2
	if f.Picks[0] != 2 || f.Chars[f.Picks[0]].Name != "c" {
		t.Fatalf("P1 pick = %d, want the roster index 2 of c", f.Picks[0])
	}
	got := f.SelectCursors()
	if got[0] != (SideCursor{Shown: true, Cursor: 1}) {
		t.Errorf("P1's cursor = %+v, want cell 1 (c) shown", got[0])
	}
	if f.Cursor != 1 {
		t.Errorf("P2 starts on cell %d, want P1's cell 1", f.Cursor)
	}
	f.Navigate(NavLeft)
	f.Navigate(NavLeft) // wraps to the last cell: e
	f.Navigate(NavConfirm)
	if f.Picks[1] != 4 || f.Screen != ScreenStage {
		t.Errorf("P2 pick = %d on screen %v, want roster index 4 (e) then the stage screen", f.Picks[1], f.Screen)
	}
	f.Navigate(NavCancel) // back to P2, whose cursor is on its pick's cell
	if f.Screen != ScreenSelect || f.Cursor != 3 {
		t.Errorf("back on the select: screen %v cursor %d, want P2 on cell 3 (e)", f.Screen, f.Cursor)
	}
}

func TestRosterFilter_StageSelect_StepsOverVisibleStagesAndPicksTheRosterIndex(t *testing.T) {
	f := rosterFlow(t, `{"stages": {"select": {"exclude": ["s1"]}}}`) // cells: s0 s2
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm)
	if f.Screen != ScreenStage {
		t.Fatalf("screen = %v", f.Screen)
	}
	f.Navigate(NavRight)
	if f.Cursor != 1 {
		t.Fatalf("right = %d, want cell 1", f.Cursor)
	}
	f.Navigate(NavRight)
	if f.Cursor != 0 {
		t.Fatalf("right from the last cell = %d, want the first", f.Cursor)
	}
	f.Navigate(NavLeft)
	if f.Cursor != 1 {
		t.Fatalf("left from the first cell = %d, want the last, 1", f.Cursor)
	}
	f.Navigate(NavConfirm) // the next step tries to start the match, which the loader refuses
	if f.StageNo != 2 {
		t.Errorf("StageNo = %d, want the roster index 2 of s2", f.StageNo)
	}
}

func TestRosterFilter_OpponentPool_DiffersFromTheSelectScreen(t *testing.T) {
	f := rosterFlow(t, `{"characters": {"select": {"exclude": ["boss"]}, "opponents": {"include": ["Boss", "c"]}}}`)
	if got := f.VisibleChars(); !slices.Equal(got, []int{0, 2, 3, 4}) {
		t.Errorf("select = %v, want the boss hidden", got)
	}
	if got := f.OpponentPool(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("opponent pool = %v, want the boss and c", got)
	}
}

func TestRosterFilter_OpponentPool_IsEveryCharacterWithoutAnOpponentsFilter(t *testing.T) {
	f := rosterFlow(t, `{"characters": {"select": {"exclude": ["boss"]}}}`)
	if got := f.OpponentPool(); !slices.Equal(got, []int{0, 1, 2, 3, 4}) {
		t.Errorf("opponent pool = %v, want every character", got)
	}
}

func TestRosterFilter_NoVisibleCharacter_ShowsAnErrorNamingTheModeAndTheListThenTheMainMenu(t *testing.T) {
	f := rosterFlow(t, `{"characters": {"select": {"include": ["nobody"]}}}`)
	if f.Screen != ScreenError {
		t.Fatalf("screen = %v, want the error screen", f.Screen)
	}
	if !strings.Contains(f.Message, `"Arcade"`) || !strings.Contains(f.Message, "roster.characters.select") {
		t.Errorf("message = %q, want it to name the mode and the list", f.Message)
	}
	if !f.ErrorReturnsToMenu() {
		t.Fatal("the error should return to the main menu")
	}
	f.Navigate(NavConfirm)
	if f.Screen != ScreenMainMenu || f.Quit {
		t.Errorf("screen = %v quit = %v, want the main menu", f.Screen, f.Quit)
	}
}

func TestRosterFilter_NoVisibleStage_ShowsAnErrorNamingTheModeAndTheList(t *testing.T) {
	f := rosterFlow(t, `{"stages": {"select": {"exclude": ["s0", "s1", "s2"]}}}`)
	if f.Screen != ScreenError || !strings.Contains(f.Message, `"Arcade"`) || !strings.Contains(f.Message, "roster.stages.select") {
		t.Errorf("screen = %v message = %q, want an error naming the mode and roster.stages.select", f.Screen, f.Message)
	}
}

func TestRosterFilter_UnknownNames_AreNoticesPerListAndNotFatal(t *testing.T) {
	m := parseArcade(t, `{"characters": {"select": {"exclude": ["Boss", "ghost"]}, "opponents": {"include": ["a", "phantom"]}}, "stages": {"select": {"include": ["S0", "nowhere"]}}}`)
	got := RosterNotices([]mode.Mode{m, versusMode(t)}, filterChars, filterStages)
	want := []string{
		`mode "Arcade": rules.roster.characters.select.exclude names "ghost", which is not an installed character`,
		`mode "Arcade": rules.roster.characters.opponents.include names "phantom", which is not an installed character`,
		`mode "Arcade": rules.roster.stages.select.include names "nowhere", which is not an installed stage`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("notices:\n%q\nwant:\n%q", got, want)
	}
	f := rosterFlow(t, `{"characters": {"select": {"exclude": ["ghost"]}}}`)
	if f.Screen != ScreenSelect {
		t.Errorf("screen = %v, want the select screen: an unknown name is not fatal", f.Screen)
	}
}

func TestRosterFilter_SortRosters_KeepsTheFilterAndThePicksOnTheirEntries(t *testing.T) {
	f := rosterFlow(t, `{"characters": {"select": {"exclude": ["boss"]}, "opponents": {"include": ["boss"]}}, "stages": {"select": {"exclude": ["s0"]}}}`)
	f.Picks[0], f.StageNo = 2, 1 // c and s1
	f.SortRosters("folder", true, "folder", true)
	// Descending: e d c boss a, and s2 s1 s0.
	if got := f.VisibleChars(); !slices.Equal(got, []int{0, 1, 2, 4}) {
		t.Errorf("visible characters = %v, want e d c a at 0 1 2 4", got)
	}
	if got := f.VisibleStages(); !slices.Equal(got, []int{0, 1}) {
		t.Errorf("visible stages = %v, want s2 s1 at 0 1", got)
	}
	if got := f.OpponentPool(); !slices.Equal(got, []int{3}) {
		t.Errorf("opponent pool = %v, want the boss at 3", got)
	}
	if f.Chars[f.Picks[0]].Name != "c" || f.Stages[f.StageNo].Name != "s1" {
		t.Errorf("picks moved to %q and %q, want c and s1", f.Chars[f.Picks[0]].Name, f.Stages[f.StageNo].Name)
	}
}

func TestRosterFilter_VersusAndWatch_ShowEveryCharacterAndStage(t *testing.T) {
	builtIns, err := mode.BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	f := NewFlow(nil, slices.Clone(filterChars), slices.Clone(filterStages), Loader{}, builtIns)
	f.enterMode(0)
	if got := f.VisibleChars(); !slices.Equal(got, []int{0, 1, 2, 3, 4}) {
		t.Errorf("versus characters = %v", got)
	}
	if got := f.VisibleStages(); !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("versus stages = %v", got)
	}
}
