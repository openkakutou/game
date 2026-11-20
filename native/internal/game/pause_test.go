package game

import (
	"testing"

	"github.com/openkakutou/engine/v2/input"
)

func pausedMatch(t *testing.T) *Flow {
	t.Helper()
	f := newTestFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm)
	if f.Screen != ScreenMatch {
		t.Fatalf("screen = %v (%s)", f.Screen, f.Message)
	}
	return f
}

func TestPause_FreezesTheMatchUntilContinue(t *testing.T) {
	f := pausedMatch(t)
	f.Tick([2]input.TickInput{})
	f.Navigate(NavCancel)
	if !f.Paused || f.PauseCursor != 0 {
		t.Fatalf("Escape must open the menu on Continue: paused=%v cursor=%d", f.Paused, f.PauseCursor)
	}
	before := f.Match.Ticks()
	for range 30 {
		f.Tick([2]input.TickInput{})
	}
	if f.Match.Ticks() != before {
		t.Fatalf("the match ran while paused: %d -> %d", before, f.Match.Ticks())
	}
	f.Navigate(NavConfirm) // Continue
	f.Tick([2]input.TickInput{})
	if f.Paused || f.Match.Ticks() != before+1 {
		t.Fatalf("Continue must resume: paused=%v ticks=%d", f.Paused, f.Match.Ticks())
	}
}

func TestPause_StartOpensAndClosesAndIsIgnoredOutsideMatches(t *testing.T) {
	f := pausedMatch(t)
	f.Navigate(NavPause)
	if !f.Paused {
		t.Fatal("Start must open the menu")
	}
	f.Navigate(NavDown)
	f.Navigate(NavPause)
	if f.Paused || f.Screen != ScreenMatch {
		t.Fatal("Start on the menu must close it, as Ikemen GO's menu.cancel.key")
	}
	f.Navigate(NavCancel)
	f.Navigate(NavCancel)
	if f.Paused {
		t.Fatal("cancel on the menu must close it")
	}
	g := newTestFlow(t)
	g.Navigate(NavPause)
	if g.Screen != ScreenSelect || g.Paused {
		t.Fatal("Start outside a match does nothing")
	}
}

func TestPause_RestartStartsTheSameMatchFresh(t *testing.T) {
	f := pausedMatch(t)
	for range 10 {
		f.Tick([2]input.TickInput{})
	}
	old, chars, st := f.Match, f.lastChars, f.lastStage
	f.Navigate(NavCancel)
	f.Navigate(NavDown)
	f.Navigate(NavConfirm)
	if f.Paused || f.Match == old || f.Match.Ticks() != 0 || f.Screen != ScreenMatch {
		t.Fatalf("restart: paused=%v sameMatch=%v ticks=%d", f.Paused, f.Match == old, f.Match.Ticks())
	}
	if f.lastChars != chars || f.lastStage != st {
		t.Errorf("restart changed the characters or stage")
	}
}

func TestPause_ExitGoesBackToTheMainMenuAndItemsDependOnTheMode(t *testing.T) {
	f := pausedMatch(t)
	items := f.PauseItems()
	if len(items) != 4 || items[2].Label() != "Character Change" || items[3].Label() != "Exit" {
		t.Fatalf("items = %v", items)
	}
	f.Navigate(NavCancel)
	f.Navigate(NavUp)
	f.Navigate(NavConfirm)
	if f.Quit || f.Screen != ScreenMainMenu || f.Paused || f.Match != nil || f.MenuCursor != 0 {
		t.Fatalf("quit=%v screen=%v paused=%v match=%v menu cursor=%d, want the main menu on Versus with no match",
			f.Quit, f.Screen, f.Paused, f.Match, f.MenuCursor)
	}
}
