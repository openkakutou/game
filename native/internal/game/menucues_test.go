package game

import "testing"

// cuesOf drains the flow's menu cues.
func cuesOf(f *Flow) []MenuCue { return f.DrainMenuCues(nil) }

func wantCues(t *testing.T, what string, got []MenuCue, want ...MenuCue) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: cues = %+v, want %+v", what, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: cues = %+v, want %+v", what, got, want)
		}
	}
}

func TestMenuCues_ConfirmOnTheTitleSoundsConfirmFromTheTitleScreen(t *testing.T) {
	f := newTitleFlow(t)
	f.Navigate(NavConfirm)
	wantCues(t, "title confirm", cuesOf(f), MenuCue{SoundConfirm, "title"})
}

func TestMenuCues_MovingTheCursorSoundsMoveAndStayingPutIsSilent(t *testing.T) {
	f := twoModeFlow(t)
	f.Navigate(NavConfirm) // the main menu: two modes, Options, Quit
	cuesOf(f)
	f.Navigate(NavDown)
	wantCues(t, "main menu down", cuesOf(f), MenuCue{SoundMove, "title"})

	g := newTestFlow(t) // two characters in one row
	cuesOf(g)
	g.Navigate(NavDown) // nowhere to go in a single row
	wantCues(t, "grid down in one row", cuesOf(g))
	g.Navigate(NavRight)
	wantCues(t, "grid right", cuesOf(g), MenuCue{SoundMove, "characterSelect"})
}

func TestMenuCues_CancelSoundsCancelOnlyWhenSomethingWasCancelled(t *testing.T) {
	f := newTestFlow(t)
	cuesOf(f)
	f.Navigate(NavCancel) // back to the main menu
	wantCues(t, "back from the select", cuesOf(f), MenuCue{SoundCancel, "characterSelect"})

	m := toMatch(t)
	cuesOf(m)
	m.Navigate(NavCancel) // a fighter's button in a running match
	wantCues(t, "cancel in a match", cuesOf(m))
}

func TestMenuCues_ChangingAValueSoundsMoveAndAValueAtItsLimitIsSilent(t *testing.T) {
	f, _ := optionsFlow(t)
	toAudio(t, f)
	cuesOf(f)
	f.Navigate(NavLeft) // Master 100 -> 90
	wantCues(t, "master down", cuesOf(f), MenuCue{SoundMove, "options"})
	f.Navigate(NavRight)
	cuesOf(f)
	f.Navigate(NavRight) // already 100
	wantCues(t, "master at its limit", cuesOf(f))
}

func TestMenuCues_ThePauseMenuSpeaksAsThePauseScreenAndOpeningItIsSilent(t *testing.T) {
	f := toMatch(t)
	cuesOf(f)
	f.Navigate(NavPause)
	wantCues(t, "opening the pause menu", cuesOf(f))
	f.Navigate(NavDown)
	wantCues(t, "pause down", cuesOf(f), MenuCue{SoundMove, "pause"})
	f.Navigate(NavConfirm) // Continue
	wantCues(t, "pause confirm", cuesOf(f), MenuCue{SoundConfirm, "pause"})
}

func TestMenuCues_AScreenThatFailsSoundsErrorFromNoScreen(t *testing.T) {
	f := newTestFlow(t)
	cuesOf(f)
	f.fail("boom")
	wantCues(t, "fail", cuesOf(f), MenuCue{SoundError, ""})
}

func TestMenuCues_ARefusedBindingAndAConflictSoundError(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	f.Navigate(NavRight) // the pad column
	cuesOf(f)
	f.Navigate(NavConfirm) // the pad has no Up
	wantCues(t, "binding a missing pad action", cuesOf(f), MenuCue{SoundError, "options"})

	f.Navigate(NavLeft) // keyboard
	onRow(t, f, "a")
	f.Navigate(NavConfirm) // waits for a key
	cuesOf(f)
	f.Bind(DeviceKeyboard, "G") // the key of b
	wantCues(t, "a conflicting key", cuesOf(f), MenuCue{SoundError, "options"})
	f.Navigate(NavCancel)
	cuesOf(f)

	f.Navigate(NavConfirm)
	cuesOf(f)
	f.Bind(DeviceKeyboard, "U") // free
	wantCues(t, "a free key", cuesOf(f), MenuCue{SoundConfirm, "options"})
}

func TestMenuCues_AnUndrainedQueueDoesNotGrowWithoutBound(t *testing.T) {
	f := twoModeFlow(t)
	f.Navigate(NavConfirm)
	for range 100 {
		f.Navigate(NavDown)
	}
	if got := len(cuesOf(f)); got != maxMenuCues {
		t.Errorf("queued cues = %d, want the cap %d", got, maxMenuCues)
	}
}

func TestMusic_MenuScreensNameTheirPackScreenAndTheMatchTakesOver(t *testing.T) {
	f := newTitleFlow(t)
	menu := func() string { _, _, m := f.Music(); return m }
	if got := menu(); got != "title" {
		t.Errorf("title = %q", got)
	}
	f.Navigate(NavConfirm)
	if got := menu(); got != "title" {
		t.Errorf("main menu = %q, want the title screen's music to go on", got)
	}
	f.Navigate(NavConfirm)
	if got := menu(); got != "characterSelect" {
		t.Errorf("select = %q", got)
	}
	f.Screen = ScreenStage
	if got := menu(); got != "stageSelect" {
		t.Errorf("stage = %q", got)
	}
	f.Screen = ScreenSetup
	if got := menu(); got != "matchSetup" {
		t.Errorf("setup = %q", got)
	}
	f.Screen = ScreenVersus
	if got := menu(); got != "versus" {
		t.Errorf("versus = %q", got)
	}
	for _, s := range []Screen{ScreenOptions, ScreenOptionsGameplay, ScreenOptionsAudio, ScreenOptionsVideo, ScreenOptionsControls, ScreenOptionsBindings} {
		f.Screen = s
		if got := menu(); got != "options" {
			t.Errorf("screen %v = %q, want options", s, got)
		}
	}
	f.Screen = ScreenError
	if got := menu(); got != "" {
		t.Errorf("error screen = %q, want no music", got)
	}
}

func TestMusic_TheStageMusicOwnsTheMatchPauseAndResultsAndMenuMusicResumesAfter(t *testing.T) {
	f := toMatch(t)
	if _, owner, menu := f.Music(); owner != f.Match || menu != "" {
		t.Errorf("match: owner %v menu %q, want the match and no menu music", owner, menu)
	}
	f.Paused = true
	if _, owner, menu := f.Music(); owner != f.Match || menu != "" {
		t.Errorf("paused match: owner %v menu %q", owner, menu)
	}
	f.Paused = false
	f.Screen = ScreenResult
	if _, owner, menu := f.Music(); owner != f.Match || menu != "" {
		t.Errorf("results: owner %v menu %q, want the stage music to go on", owner, menu)
	}
	f.Navigate(NavCancel) // back to the select
	if _, owner, menu := f.Music(); owner != nil || menu != "characterSelect" {
		t.Errorf("after the results: owner %v menu %q, want the menu music back", owner, menu)
	}
}
