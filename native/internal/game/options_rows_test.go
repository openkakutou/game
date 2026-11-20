package game

import (
	"testing"

	"github.com/openkakutou/engine/v2/match"
)

func TestOptionsRows_SplitEachRowIntoItsLabelAndItsValues(t *testing.T) {
	f, _ := optionsFlow(t)
	toAudio(t, f)
	want := []OptionRow{
		{Label: "Master", Values: [2]string{"100", ""}},
		{Label: "Music", Values: [2]string{"100", ""}},
		{Label: "Effects", Values: [2]string{"100", ""}},
		{Label: "Back"},
	}
	if got := f.OptionsRows(); !sameRows(got, want) {
		t.Fatalf("rows = %+v, want %+v", got, want)
	}
	navAll(f, NavDown, NavLeft)
	if got := f.OptionsRows()[1]; got.Values[0] != "90" {
		t.Errorf("music row after left = %+v, want value 90", got)
	}
}

func TestOptionsRows_TheConcernsListHasLabelsOnly(t *testing.T) {
	f, _ := optionsFlow(t)
	navAll(f, NavDown, NavConfirm)
	got := f.OptionsRows()
	if len(got) != 6 || got[3].Label != "Controls" || got[3].Values != [2]string{} {
		t.Errorf("rows = %+v, want Gameplay, Audio, Video, Controls, Appearance, Back as bare labels", got)
	}
}

func TestOptionsRows_BindingRowsCarryTheKeyThenTheButtonAndTheFlaggedMark(t *testing.T) {
	f, _ := controlsFlow(t, map[string]string{"controls.json": `{"linux": {"p1": {"keyboard": {"a": "G"}}}}`})
	toControls(t, f)
	toPlayer(t, f, 0)
	row := f.OptionsRows()[4] // the action "a": its key G is also the key of "b"
	if row.Label != "A" || row.Values != [2]string{"G", "a"} || !row.Flagged {
		t.Errorf("row = %+v, want A, G, a, flagged", row)
	}
	if f.OptionsRows()[0].Values != [2]string{"W", "-"} {
		t.Errorf("up row = %+v, want W and no button", f.OptionsRows()[0])
	}
	if f.OptionsColumn() != 0 {
		t.Errorf("column = %d, want the keyboard's", f.OptionsColumn())
	}
	f.Navigate(NavRight)
	if f.OptionsColumn() != 1 {
		t.Errorf("column = %d after right, want the pad's", f.OptionsColumn())
	}
}

func sameRows(a, b []OptionRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestResultMenu_OffersRematchAndBackWithACursor(t *testing.T) {
	f := newTestFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavRight)
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm) // stage
	f.Navigate(NavConfirm) // setup
	f.Navigate(NavConfirm) // VS
	f.Navigate(NavConfirm)
	finishMatch(t, f, match.SideP1)
	if want := []string{"Rematch", "Back to selection"}; !sameStrings(f.ResultItems(), want) || f.ResultCursor != 0 {
		t.Fatalf("items = %q cursor = %d, want %q and 0", f.ResultItems(), f.ResultCursor, want)
	}
	f.Navigate(NavDown)
	if f.ResultCursor != 1 {
		t.Fatalf("cursor = %d after down, want 1", f.ResultCursor)
	}
	f.Navigate(NavDown)
	if f.ResultCursor != 0 {
		t.Errorf("cursor = %d after a second down, want it to wrap to 0", f.ResultCursor)
	}
	f.Navigate(NavUp)
	f.Navigate(NavConfirm) // Back to selection
	if f.Screen != ScreenSelect {
		t.Errorf("screen = %v, want the character select", f.Screen)
	}
}
