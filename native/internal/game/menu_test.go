package game

import (
	"testing"

	"github.com/openkakutou/game/native/internal/mode"
)

// twoModeFlow is a flow on the title with two modes, Arcade then Versus.
func twoModeFlow(t *testing.T) *Flow {
	t.Helper()
	arcade := versusMode(t)
	arcade.ID, arcade.Name = "arcade", "Arcade"
	f := newTitleFlow(t)
	return NewFlow(f.FS, f.Chars, f.Stages, f.Loader, []mode.Mode{arcade, versusMode(t)})
}

func TestTitle_StartsTheFlowAndEntersTheMainMenuOnConfirmOrStart(t *testing.T) {
	for _, n := range []Nav{NavConfirm, NavPause} {
		f := newTitleFlow(t)
		if f.Screen != ScreenTitle {
			t.Fatalf("screen = %v, want the title", f.Screen)
		}
		f.Navigate(n)
		if f.Screen != ScreenMainMenu {
			t.Errorf("nav %v: screen = %v, want the main menu", n, f.Screen)
		}
	}
}

func TestTitle_CancelQuitsAndMovesAreIgnored(t *testing.T) {
	f := newTitleFlow(t)
	f.Navigate(NavDown)
	if f.Screen != ScreenTitle || f.Quit {
		t.Fatalf("screen = %v quit = %v, want the title untouched", f.Screen, f.Quit)
	}
	f.Navigate(NavCancel)
	if !f.Quit {
		t.Error("cancel on the title should quit")
	}
}

func TestMainMenu_ListsTheModeNamesInOrderThenQuit(t *testing.T) {
	f := twoModeFlow(t)
	got := f.MainMenuItems()
	if want := []string{"Arcade", "Versus", "Quit"}; !sameStrings(got, want) {
		t.Fatalf("items = %q, want %q", got, want)
	}
}

func TestMainMenu_CursorWrapsBothWays(t *testing.T) {
	f := twoModeFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavUp)
	if f.MenuCursor != 2 {
		t.Fatalf("cursor = %d, want wrap to Quit (2)", f.MenuCursor)
	}
	f.Navigate(NavDown)
	if f.MenuCursor != 0 {
		t.Errorf("cursor = %d, want wrap to 0", f.MenuCursor)
	}
}

func TestMainMenu_ConfirmOnAModeEntersItsFirstStep(t *testing.T) {
	f := twoModeFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavDown) // Versus
	f.Navigate(NavConfirm)
	if f.Mode.ID != "versus" || f.Screen != ScreenSelect || f.Picking != 0 {
		t.Fatalf("mode = %q screen = %v picking = %d, want Versus on its character select", f.Mode.ID, f.Screen, f.Picking)
	}
}

func TestMainMenu_ConfirmOnQuitQuits(t *testing.T) {
	f := newTitleFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavDown) // Quit
	f.Navigate(NavConfirm)
	if !f.Quit {
		t.Error("Quit item should quit")
	}
}

func TestMainMenu_CancelGoesBackToTheTitle(t *testing.T) {
	f := newTitleFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavCancel)
	if f.Screen != ScreenTitle || f.Quit {
		t.Errorf("screen = %v quit = %v, want the title", f.Screen, f.Quit)
	}
}

func TestMainMenu_CancelOnTheFirstStepReturnsWithTheCursorOnThatMode(t *testing.T) {
	f := twoModeFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavDown)
	f.Navigate(NavConfirm) // Versus
	f.Navigate(NavCancel)
	if f.Screen != ScreenMainMenu || f.MenuCursor != 1 {
		t.Errorf("screen = %v cursor = %d, want the main menu on Versus (1)", f.Screen, f.MenuCursor)
	}
}

func TestMainMenu_ResultsCancelWithoutCharacterSelectReturnsToTheMenu(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(stepStage, stepMatch, stepResult), "human", "human", 1, 99, "results", "results"))
	f.Navigate(NavConfirm) // stage
	finishMatch(t, f, 0)
	f.Navigate(NavCancel)
	if f.Screen != ScreenMainMenu || f.Quit || f.Match != nil {
		t.Errorf("screen = %v quit = %v match = %v, want the main menu", f.Screen, f.Quit, f.Match)
	}
}

func TestMainMenu_EnteringAModeAgainStartsFresh(t *testing.T) {
	f := newTestFlow(t)
	f.Navigate(NavRight)
	f.Navigate(NavConfirm) // P1 = 1
	f.Navigate(NavRight)
	f.Navigate(NavConfirm) // P2 = 0
	f.Navigate(NavDown)
	f.Navigate(NavConfirm) // stage 1
	f.SetupRow = SetupRowRounds
	f.Navigate(NavRight)  // rounds 5
	f.Navigate(NavCancel) // stage select
	f.Navigate(NavCancel) // P2 picking
	f.Navigate(NavCancel) // P1 picking
	f.Navigate(NavCancel) // main menu
	if f.Screen != ScreenMainMenu {
		t.Fatalf("screen = %v, want the main menu", f.Screen)
	}
	f.Navigate(NavConfirm)
	if f.Screen != ScreenSelect || f.Picks != [2]int{0, 0} || f.StageNo != 0 || f.Cursor != 0 {
		t.Errorf("screen = %v picks = %v stage = %d cursor = %d, want a fresh select", f.Screen, f.Picks, f.StageNo, f.Cursor)
	}
	if s := f.Setup(); s.Rounds != 3 {
		t.Errorf("rounds = %d, want the rules' 3 again", s.Rounds)
	}
}
