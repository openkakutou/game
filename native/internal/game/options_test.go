package game

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/mode"
)

// optionsFlow is the Versus flow on its main menu, over default settings
// whose first-launch files were already written to rec.
func optionsFlow(t *testing.T) (*Flow, *recorder) {
	t.Helper()
	rec := &recorder{}
	f := newTitleFlow(t)
	f.Settings = LoadSettings(fstest.MapFS{}, "linux", rec.write, nil)
	rec.names = nil
	f.Navigate(NavConfirm) // main menu
	return f, rec
}

func navAll(f *Flow, ns ...Nav) {
	for _, n := range ns {
		f.Navigate(n)
	}
}

// toAudio opens Options > Audio from the main menu.
func toAudio(t *testing.T, f *Flow) {
	t.Helper()
	navAll(f, NavDown, NavConfirm, NavDown, NavConfirm)
	if f.Screen != ScreenOptionsAudio {
		t.Fatalf("screen = %v, want the audio options", f.Screen)
	}
}

func TestMainMenu_ShowsOptionsBeforeQuit(t *testing.T) {
	f := twoModeFlow(t)
	if want := []string{"Arcade", "Versus", "Options", "Quit"}; !sameStrings(f.MainMenuItems(), want) {
		t.Fatalf("items = %q, want %q", f.MainMenuItems(), want)
	}
}

func TestOptions_AudioChangeIsSavedAloneWhenLeavingTheSubScreen(t *testing.T) {
	f, rec := optionsFlow(t)
	toAudio(t, f)
	if want := []string{"Master  100", "Music  100", "Effects  100", "Back"}; !sameStrings(f.OptionsItems(), want) {
		t.Fatalf("rows = %q, want %q", f.OptionsItems(), want)
	}
	gen := f.Settings.AudioGen
	navAll(f, NavDown, NavLeft)
	if f.Settings.Audio != (Audio{100, 90, 100}) || f.OptionsItems()[1] != "Music  90" {
		t.Fatalf("audio = %+v rows = %q", f.Settings.Audio, f.OptionsItems())
	}
	if f.Settings.AudioGen == gen {
		t.Error("audio change must be visible to the main loop at once")
	}
	if len(rec.names) != 0 {
		t.Fatalf("written %q before leaving the sub-screen", rec.names)
	}
	navAll(f, NavDown, NavDown, NavConfirm) // Back row
	if f.Screen != ScreenOptions || f.OptionsCursor() != 1 {
		t.Errorf("screen = %v cursor = %d, want Options on Audio", f.Screen, f.OptionsCursor())
	}
	if len(rec.names) != 1 || rec.names[0] != "config/audio.json" {
		t.Fatalf("written = %q, want exactly config/audio.json", rec.names)
	}
	got := decodeJSON(t, rec.files["config/audio.json"])
	if got["music"] != 90.0 || got["master"] != 100.0 || got["effects"] != 100.0 {
		t.Errorf("audio.json = %v", got)
	}
}

func TestOptions_CancelLeavesASubScreenAndSavesItToo(t *testing.T) {
	f, rec := optionsFlow(t)
	toAudio(t, f)
	navAll(f, NavLeft, NavCancel)
	if f.Screen != ScreenOptions || len(rec.names) != 1 || rec.names[0] != "config/audio.json" {
		t.Errorf("screen = %v written = %q", f.Screen, rec.names)
	}
}

func TestOptions_AnUnchangedSubScreenSavesNothing(t *testing.T) {
	f, rec := optionsFlow(t)
	navAll(f, NavDown, NavConfirm, NavConfirm) // Options > Gameplay
	if f.Screen != ScreenOptionsGameplay {
		t.Fatalf("screen = %v", f.Screen)
	}
	navAll(f, NavCancel)
	if f.Screen != ScreenOptions || len(rec.names) != 0 {
		t.Errorf("screen = %v written = %q, want nothing", f.Screen, rec.names)
	}
}

func TestOptions_AVolumeAtItsLimitDoesNotChangeNorSave(t *testing.T) {
	f, rec := optionsFlow(t)
	toAudio(t, f)
	gen := f.Settings.AudioGen
	navAll(f, NavRight, NavCancel)
	if f.Settings.Audio.Master != 100 || f.Settings.AudioGen != gen || len(rec.names) != 0 {
		t.Errorf("master = %d gen %d->%d written %q", f.Settings.Audio.Master, gen, f.Settings.AudioGen, rec.names)
	}
}

func TestOptions_CancelOnOptionsReturnsToTheMainMenuOnOptions(t *testing.T) {
	f, _ := optionsFlow(t)
	navAll(f, NavDown, NavConfirm, NavCancel)
	if f.Screen != ScreenMainMenu || f.MenuCursor != 1 {
		t.Errorf("screen = %v cursor = %d", f.Screen, f.MenuCursor)
	}
}

func TestOptions_OpenedFromThePauseMenuReturnsToThePausedMatch(t *testing.T) {
	f := pausedMatch(t)
	navAll(f, NavCancel, NavDown, NavDown, NavDown) // pause menu, Options
	if got := f.PauseItems()[f.PauseCursor].Label(); got != "Options" {
		t.Fatalf("pause cursor on %q", got)
	}
	m := f.Match
	f.Navigate(NavConfirm)
	if f.Screen != ScreenOptions {
		t.Fatalf("screen = %v, want Options", f.Screen)
	}
	if _, owner := f.Music(); owner != m {
		t.Error("the match must keep the music while Options is open over it")
	}
	f.Navigate(NavCancel)
	if f.Screen != ScreenMatch || !f.Paused || f.Match != m || f.PauseCursor != 3 {
		t.Errorf("screen = %v paused = %v sameMatch = %v cursor = %d", f.Screen, f.Paused, f.Match == m, f.PauseCursor)
	}
}

func TestPause_QuickVersusMenuHasOptionsBeforeExit(t *testing.T) {
	f := &Flow{quick: &Setup{}}
	var labels []string
	for _, it := range f.PauseItems() {
		labels = append(labels, it.Label())
	}
	if want := []string{"Continue", "Restart", "Options", "Exit"}; !sameStrings(labels, want) {
		t.Errorf("labels = %q, want %q", labels, want)
	}
}

func TestOptions_GameplaySettingsPreselectTheVersusSetup(t *testing.T) {
	f, _ := optionsFlow(t)
	navAll(f, NavDown, NavConfirm, NavConfirm, NavRight, NavCancel, NavCancel, NavUp, NavConfirm) // rounds 3 -> 5
	if f.Settings.Gameplay.Rounds != 5 {
		t.Fatalf("rounds = %d", f.Settings.Gameplay.Rounds)
	}
	navAll(f, NavConfirm, NavConfirm, NavConfirm) // P1, P2, stage
	if f.Screen != ScreenSetup {
		t.Fatalf("screen = %v", f.Screen)
	}
	if got := f.Setup(); got.Rounds != 5 || got.TimeLimitSeconds != 99 {
		t.Errorf("setup = %+v, want 5 rounds and 99 seconds", got)
	}
}

func TestOptions_TimeLimitOutsideTheStockListIsOfferedAndPreselected(t *testing.T) {
	f := newTitleFlow(t)
	f.Settings.Gameplay = Gameplay{Rounds: 7, TimeLimitSeconds: 45}
	inMode(t, f)
	navAll(f, NavConfirm, NavConfirm, NavConfirm)
	if got := f.Setup(); got.Rounds != 7 || got.TimeLimitSeconds != 45 {
		t.Errorf("setup = %+v, want 7 rounds and 45 seconds", got)
	}
}

func TestOptions_AModeWithoutMatchSetupKeepsItsOwnRules(t *testing.T) {
	m, err := mode.Parse("t.jsonc", []byte(modeSrc(steps(stepSelect, stepStage, stepMatch, stepResult), "human", "human", 1, 60, "results", "results")))
	if err != nil {
		t.Fatal(err)
	}
	f := newTitleFlow(t)
	f.Modes = []mode.Mode{m}
	f.Settings.Gameplay = Gameplay{Rounds: 5, TimeLimitSeconds: 30}
	inMode(t, f)
	if got := f.Setup(); got.Rounds != 1 || got.TimeLimitSeconds != 60 {
		t.Errorf("setup = %+v, want the rules' 1 round and 60 seconds", got)
	}
}

func TestOptions_AWriteFailureIsANoticeOnTheOptionsScreen(t *testing.T) {
	f, rec := optionsFlow(t)
	rec.err = errors.New("read-only folder")
	toAudio(t, f)
	navAll(f, NavLeft, NavCancel)
	if f.Screen != ScreenOptions {
		t.Fatalf("screen = %v, the game must keep running", f.Screen)
	}
	if n := f.Settings.Notices(); len(n) != 1 || !strings.Contains(n[0], "config/audio.json") {
		t.Errorf("notices = %q", n)
	}
	if f.Settings.Audio.Master != 90 {
		t.Errorf("master = %d, the change must still apply", f.Settings.Audio.Master)
	}
}

func TestOptions_VideoRowsToggleAndCycleTheWindowSize(t *testing.T) {
	f, rec := optionsFlow(t)
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavConfirm) // Options > Video
	if f.Screen != ScreenOptionsVideo {
		t.Fatalf("screen = %v", f.Screen)
	}
	if want := []string{"Fullscreen  Off", "VSync  On", "Window  960x720", "Back"}; !sameStrings(f.OptionsItems(), want) {
		t.Fatalf("rows = %q, want %q", f.OptionsItems(), want)
	}
	gen := f.Settings.VideoGen
	navAll(f, NavRight, NavDown, NavLeft, NavDown, NavRight)
	if f.Settings.Video != (Video{true, false, 1280, 960}) || f.Settings.VideoGen == gen {
		t.Errorf("video = %+v", f.Settings.Video)
	}
	f.Navigate(NavCancel)
	if len(rec.names) != 1 || rec.names[0] != "config/video.json" {
		t.Errorf("written = %q", rec.names)
	}
}

func TestOptions_ACustomWindowSizeStaysInTheSortedList(t *testing.T) {
	f, _ := optionsFlow(t)
	f.Settings.Video.WindowWidth, f.Settings.Video.WindowHeight = 1000, 800
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavConfirm, NavDown, NavDown, NavRight)
	if got := f.Settings.Video; got.WindowWidth != 1280 || got.WindowHeight != 960 {
		t.Fatalf("after right: %dx%d, want 1280x960", got.WindowWidth, got.WindowHeight)
	}
	f.Settings.Video.WindowWidth, f.Settings.Video.WindowHeight = 1000, 800
	f.refreshRows()
	f.Navigate(NavLeft)
	if got := f.Settings.Video; got.WindowWidth != 960 || got.WindowHeight != 720 {
		t.Errorf("left from 1000x800: %dx%d, want 960x720", got.WindowWidth, got.WindowHeight)
	}
}
