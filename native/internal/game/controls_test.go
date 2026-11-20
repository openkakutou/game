package game

import (
	"errors"
	"testing"
	"testing/fstest"
)

// toControls opens Options > Controls from the main menu.
func toControls(t *testing.T, f *Flow) {
	t.Helper()
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavDown, NavConfirm)
	if f.Screen != ScreenOptionsControls {
		t.Fatalf("screen = %v, want the controls menu", f.Screen)
	}
}

// toPlayer opens the bindings of player (0 or 1) from the controls menu.
func toPlayer(t *testing.T, f *Flow, player int) {
	t.Helper()
	for range player {
		f.Navigate(NavDown)
	}
	f.Navigate(NavConfirm)
	if f.Screen != ScreenOptionsBindings {
		t.Fatalf("screen = %v, want the bindings screen", f.Screen)
	}
}

// onRow moves the cursor of the bindings screen to the row of action.
func onRow(t *testing.T, f *Flow, action string) {
	t.Helper()
	for i, a := range bindRows {
		if a == action {
			for f.OptionsCursor() < i {
				f.Navigate(NavDown)
			}
			return
		}
	}
	t.Fatalf("no row for %q", action)
}

// controlsFlow is optionsFlow over a settings folder holding files.
func controlsFlow(t *testing.T, files map[string]string) (*Flow, *recorder) {
	t.Helper()
	rec := &recorder{}
	f := newTitleFlow(t)
	f.Settings = LoadSettings(configFS(files), "linux", rec.write, nil)
	f.Navigate(NavConfirm)
	return f, rec
}

func TestControls_DefaultsAreTheBindingsOfTheWebBuild(t *testing.T) {
	c := NewSettings().Controls
	wantKeys := [2][10]string{
		{"W", "S", "A", "D", "F", "G", "H", "R", "T", "Y"},
		{"Up", "Down", "Left", "Right", "J", "K", "L", "I", "O", "P"},
	}
	wantPad := [7]string{"a", "b", "x", "y", "leftshoulder", "rightshoulder", "start"}
	for p := range c {
		if c[p].Keyboard != wantKeys[p] {
			t.Errorf("player %d keyboard = %q, want %q", p+1, c[p].Keyboard, wantKeys[p])
		}
		if c[p].Pad != wantPad {
			t.Errorf("player %d pad = %q, want %q", p+1, c[p].Pad, wantPad)
		}
	}
	if want := [10]string{"up", "down", "left", "right", "a", "b", "c", "x", "y", "z"}; KeyboardActions != want {
		t.Errorf("keyboard actions = %q", KeyboardActions)
	}
	if want := [7]string{"a", "b", "c", "x", "y", "z", "start"}; PadActions != want {
		t.Errorf("pad actions = %q", PadActions)
	}
}

func TestControls_FirstLaunchWritesDefaultsForThePlatform(t *testing.T) {
	rec := &recorder{}
	LoadSettings(fstest.MapFS{}, "linux", rec.write, nil)
	got := decodeJSON(t, rec.files["config/controls.json"])
	p1 := got["linux"].(map[string]any)["p1"].(map[string]any)
	kb := p1["keyboard"].(map[string]any)
	pad := p1["gamepad"].(map[string]any)
	if kb["up"] != "W" || kb["a"] != "F" || kb["z"] != "Y" || len(kb) != 10 {
		t.Errorf("p1 keyboard = %v", kb)
	}
	if pad["c"] != "x" || pad["start"] != "start" || len(pad) != 7 {
		t.Errorf("p1 gamepad = %v", pad)
	}
	p2 := got["linux"].(map[string]any)["p2"].(map[string]any)
	if p2["keyboard"].(map[string]any)["left"] != "Left" {
		t.Errorf("p2 keyboard = %v", p2["keyboard"])
	}
}

func TestControls_LoadsTheRunningPlatformsSectionKeepingDefaultsForMissingActions(t *testing.T) {
	s := LoadSettings(configFS(map[string]string{"controls.json": `{
		"linux": {"p1": {"keyboard": {"a": "U"}, "gamepad": {"start": "back"}}},
		"windows": {"p1": {"keyboard": {"a": "Q"}}}}`}), "linux", nil, nil)
	if s.Controls[0].Keyboard[4] != "U" || s.Controls[0].Keyboard[5] != "G" {
		t.Errorf("p1 keyboard = %q, want a=U and b=G", s.Controls[0].Keyboard)
	}
	if s.Controls[0].Pad[6] != "back" || s.Controls[0].Pad[0] != "a" {
		t.Errorf("p1 pad = %q, want start=back and a=a", s.Controls[0].Pad)
	}
	if s.Controls[1] != NewSettings().Controls[1] {
		t.Errorf("p2 = %+v, want the defaults", s.Controls[1])
	}
	if len(s.Notices()) != 0 {
		t.Errorf("notices = %q", s.Notices())
	}
}

func TestControls_InvalidFileFallsBackToDefaultsWithANoticeForControlsOnly(t *testing.T) {
	bad := map[string]string{
		"unknown action":  `{"linux": {"p1": {"keyboard": {"jump": "J"}}}}`,
		"empty name":      `{"linux": {"p1": {"keyboard": {"a": ""}}}}`,
		"keyboard action": `{"linux": {"p1": {"keyboard": {"start": "S"}}}}`,
		"pad direction":   `{"linux": {"p1": {"gamepad": {"up": "dpup"}}}}`,
		"unknown device":  `{"linux": {"p1": {"mouse": {"a": "x"}}}}`,
		"unknown player":  `{"linux": {"p3": {}}}`,
		"section":         `{"linux": 3}`,
	}
	for name, body := range bad {
		rec := &recorder{}
		var warned []string
		s := LoadSettings(configFS(map[string]string{
			"controls.json": body, "audio.json": `{"master": 40, "music": 50, "effects": 60}`,
		}), "linux", rec.write, func(m string) { warned = append(warned, m) })
		if s.Controls != NewSettings().Controls {
			t.Errorf("%s: controls = %+v, want the defaults", name, s.Controls)
		}
		if s.Audio != (Audio{40, 50, 60}) {
			t.Errorf("%s: audio = %+v, want the file's values", name, s.Audio)
		}
		if want := []string{"config/controls.json: defaults used"}; !sameStrings(s.Notices(), want) {
			t.Errorf("%s: notices = %q, want %q", name, s.Notices(), want)
		}
		if len(warned) != 1 {
			t.Errorf("%s: warned %q, want one message", name, warned)
		}
		if _, written := rec.files["config/controls.json"]; written {
			t.Errorf("%s: the invalid file was overwritten at load", name)
		}
	}
}

func TestControls_ABindingSharedByTwoActionsInAFileIsKeptNotCorrupt(t *testing.T) {
	s := LoadSettings(configFS(map[string]string{"controls.json": `{"linux": {"p1": {"keyboard": {"a": "G"}}}}`}), "linux", nil, nil)
	if s.Controls[0].Keyboard[4] != "G" || s.Controls[0].Keyboard[5] != "G" || len(s.Notices()) != 0 {
		t.Errorf("keyboard = %q notices = %q", s.Controls[0].Keyboard, s.Notices())
	}
}

func TestControls_RejectedNamesGiveDefaultsANoticeAndANewGeneration(t *testing.T) {
	s := LoadSettings(configFS(map[string]string{"controls.json": `{"linux": {"p1": {"keyboard": {"a": "Nope"}}}}`}), "linux", nil, nil)
	gen := s.ControlsGen
	s.RejectControls(errors.New(`unknown key "Nope"`))
	if s.Controls != NewSettings().Controls {
		t.Errorf("controls = %+v, want the defaults", s.Controls)
	}
	if s.ControlsGen == gen {
		t.Error("the generation must grow so the host resolves the defaults")
	}
	if want := []string{"config/controls.json: defaults used"}; !sameStrings(s.Notices(), want) {
		t.Errorf("notices = %q, want %q", s.Notices(), want)
	}
}

func TestOptions_ListsControlsBeforeBack(t *testing.T) {
	f, _ := optionsFlow(t)
	navAll(f, NavDown, NavConfirm)
	if want := []string{"Gameplay", "Audio", "Video", "Controls", "Back"}; !sameStrings(f.OptionsItems(), want) {
		t.Fatalf("items = %q, want %q", f.OptionsItems(), want)
	}
}

func TestControlsMenu_ListsPlayersResetAndBack(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	if want := []string{"Player 1", "Player 2", "Reset to defaults", "Back"}; !sameStrings(f.OptionsItems(), want) {
		t.Errorf("items = %q, want %q", f.OptionsItems(), want)
	}
	if f.OptionsTitle() != "Controls" {
		t.Errorf("title = %q", f.OptionsTitle())
	}
}

func TestBindingsScreen_ListsEveryActionWithItsKeyAndButtonAndTheColumnInBrackets(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	want := []string{
		"Up  <W>  -", "Down  <S>  -", "Left  <A>  -", "Right  <D>  -",
		"A  <F>  a", "B  <G>  b", "C  <H>  x", "X  <R>  y", "Y  <T>  leftshoulder", "Z  <Y>  rightshoulder",
		"Start  <->  start", "Back",
	}
	if !sameStrings(f.OptionsItems(), want) {
		t.Errorf("rows = %q, want %q", f.OptionsItems(), want)
	}
	if f.OptionsTitle() != "Player 1 controls" {
		t.Errorf("title = %q", f.OptionsTitle())
	}
	f.Navigate(NavRight)
	if got := f.OptionsItems()[4]; got != "A  F  <a>" {
		t.Errorf("row after right = %q, want %q", got, "A  F  <a>")
	}
	f.Navigate(NavLeft)
	if got := f.OptionsItems()[4]; got != "A  <F>  a" {
		t.Errorf("row after left = %q, want %q", got, "A  <F>  a")
	}
}

func TestBindingsScreen_RowsSharingABindingAreFlaggedRed(t *testing.T) {
	f, _ := controlsFlow(t, map[string]string{"controls.json": `{"linux": {"p1": {"keyboard": {"a": "G"}}}}`})
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavDown, NavConfirm)
	toPlayer(t, f, 0)
	flagged := f.OptionsFlagged()
	for i, a := range bindRows {
		want := a == "a" || a == "b"
		if flagged[i] != want {
			t.Errorf("row %q flagged = %v, want %v", a, flagged[i], want)
		}
	}
}

func TestBindingsScreen_AKeyHeldByTheOtherPlayerFlagsBothRows(t *testing.T) {
	f, _ := controlsFlow(t, map[string]string{"controls.json": `{"linux": {"p2": {"keyboard": {"a": "F"}}}}`})
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavDown, NavConfirm)
	toPlayer(t, f, 0)
	if !f.OptionsFlagged()[4] {
		t.Error("P1 a (F) shares its key with P2 a and must be flagged")
	}
	if f.OptionsFlagged()[5] {
		t.Error("P1 b (G) is not shared and must not be flagged")
	}
}

func TestBind_AFreeKeyIsAssignedAndSavedOnceWhenLeavingTheControls(t *testing.T) {
	f, rec := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	onRow(t, f, "a")
	gen := f.Settings.ControlsGen
	f.Navigate(NavConfirm)
	if dev, ok := f.WaitingForBinding(); !ok || dev != DeviceKeyboard {
		t.Fatalf("waiting = %v %v, want the keyboard", dev, ok)
	}
	if f.OptionsPrompt() != "Press a key..." {
		t.Errorf("prompt = %q", f.OptionsPrompt())
	}
	f.Bind(DeviceKeyboard, "U")
	if _, ok := f.WaitingForBinding(); ok {
		t.Error("still waiting after the key")
	}
	if f.Settings.Controls[0].Keyboard[4] != "U" || f.OptionsItems()[4] != "A  <U>  a" {
		t.Errorf("keyboard = %q row = %q", f.Settings.Controls[0].Keyboard, f.OptionsItems()[4])
	}
	if f.Settings.ControlsGen != gen+1 {
		t.Errorf("generation = %d, want %d", f.Settings.ControlsGen, gen+1)
	}
	if f.OptionsPrompt() != "" {
		t.Errorf("prompt = %q after the bind", f.OptionsPrompt())
	}
	if len(rec.names) != 0 {
		t.Fatalf("written %q before leaving", rec.names)
	}
	navAll(f, NavCancel) // back to the controls menu
	if len(rec.names) != 0 || f.Screen != ScreenOptionsControls || f.OptionsCursor() != 0 {
		t.Fatalf("screen = %v cursor = %d written = %q", f.Screen, f.OptionsCursor(), rec.names)
	}
	navAll(f, NavCancel) // back to Options: saves
	if len(rec.names) != 1 || rec.names[0] != "config/controls.json" {
		t.Fatalf("written = %q, want exactly config/controls.json", rec.names)
	}
	kb := decodeJSON(t, rec.files["config/controls.json"])["linux"].(map[string]any)["p1"].(map[string]any)["keyboard"].(map[string]any)
	if kb["a"] != "U" || kb["b"] != "G" {
		t.Errorf("saved keyboard = %v", kb)
	}
	if f.Screen != ScreenOptions || f.OptionsCursor() != 3 {
		t.Errorf("screen = %v cursor = %d, want Options on Controls", f.Screen, f.OptionsCursor())
	}
}

func TestBind_PadButtonsAreBoundInTheGamepadColumn(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 1)
	f.Navigate(NavRight)
	onRow(t, f, "start")
	f.Navigate(NavConfirm)
	if dev, ok := f.WaitingForBinding(); !ok || dev != DevicePad {
		t.Fatalf("waiting = %v %v, want the pad", dev, ok)
	}
	if f.OptionsPrompt() != "Press a button..." {
		t.Errorf("prompt = %q", f.OptionsPrompt())
	}
	f.Bind(DeviceKeyboard, "U") // a key while a button is awaited
	if f.Settings.Controls[1].Keyboard != NewSettings().Controls[1].Keyboard {
		t.Fatal("a key must not bind while a button is awaited")
	}
	f.Bind(DevicePad, "") // a button SDL has no name for
	if _, ok := f.WaitingForBinding(); !ok {
		t.Fatal("an empty name must not end the wait")
	}
	f.Bind(DevicePad, "back")
	if f.Settings.Controls[1].Pad[6] != "back" || f.Settings.Controls[0].Pad[6] != "start" {
		t.Errorf("pads = %q / %q", f.Settings.Controls[1].Pad, f.Settings.Controls[0].Pad)
	}
}

func TestBind_ColumnsWithoutAnActionDoNotWait(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	f.Navigate(NavRight) // pad column, on Up: directions stay on the d-pad
	f.Navigate(NavConfirm)
	if _, ok := f.WaitingForBinding(); ok {
		t.Error("Up has no pad button, it must not wait")
	}
	f.Navigate(NavLeft)
	onRow(t, f, "start")
	f.Navigate(NavConfirm)
	if _, ok := f.WaitingForBinding(); ok {
		t.Error("Start has no key, it must not wait")
	}
}

func TestBind_EscapeWhileWaitingCancelsWithoutChangingAnything(t *testing.T) {
	f, rec := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	onRow(t, f, "a")
	gen := f.Settings.ControlsGen
	f.Navigate(NavConfirm)
	navAll(f, NavUp, NavDown, NavRight, NavConfirm, NavPause) // everything else is ignored
	if _, ok := f.WaitingForBinding(); !ok || f.OptionsCursor() != 4 {
		t.Fatalf("waiting must go on, cursor = %d", f.OptionsCursor())
	}
	f.Navigate(NavCancel)
	if _, ok := f.WaitingForBinding(); ok || f.Screen != ScreenOptionsBindings {
		t.Errorf("screen = %v: the cancel must end the wait and stay on the screen", f.Screen)
	}
	if f.Settings.Controls != NewSettings().Controls || f.Settings.ControlsGen != gen {
		t.Error("the cancel changed the bindings")
	}
	navAll(f, NavCancel, NavCancel)
	if len(rec.names) != 0 {
		t.Errorf("written %q, nothing changed", rec.names)
	}
}

func TestBind_AKeyOfTheOtherPlayerIsAConflictThatNamesItAndChangesNothingYet(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	onRow(t, f, "a")
	gen := f.Settings.ControlsGen
	f.Navigate(NavConfirm)
	f.Bind(DeviceKeyboard, "J") // P2's a
	if want := "J is Player 2 A: Enter swaps, Esc keeps"; f.OptionsPrompt() != want {
		t.Errorf("prompt = %q, want %q", f.OptionsPrompt(), want)
	}
	if f.Settings.Controls != NewSettings().Controls || f.Settings.ControlsGen != gen {
		t.Error("a conflict must not change anything before the answer")
	}
	if _, ok := f.WaitingForBinding(); ok {
		t.Error("the conflict question is not the wait for a key")
	}
}

func TestBind_ConfirmingAConflictSwapsBothBindings(t *testing.T) {
	f, rec := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	onRow(t, f, "a")
	gen := f.Settings.ControlsGen
	f.Navigate(NavConfirm)
	f.Bind(DeviceKeyboard, "J")
	f.Navigate(NavConfirm)
	c := f.Settings.Controls
	if c[0].Keyboard[4] != "J" || c[1].Keyboard[4] != "F" {
		t.Errorf("p1 a = %q p2 a = %q, want J and F", c[0].Keyboard[4], c[1].Keyboard[4])
	}
	if f.OptionsItems()[4] != "A  <J>  a" || f.OptionsPrompt() != "" || f.Settings.ControlsGen != gen+1 {
		t.Errorf("row = %q prompt = %q gen = %d", f.OptionsItems()[4], f.OptionsPrompt(), f.Settings.ControlsGen)
	}
	navAll(f, NavCancel, NavCancel)
	saved := decodeJSON(t, rec.files["config/controls.json"])["linux"].(map[string]any)
	if saved["p1"].(map[string]any)["keyboard"].(map[string]any)["a"] != "J" ||
		saved["p2"].(map[string]any)["keyboard"].(map[string]any)["a"] != "F" {
		t.Errorf("saved = %v", saved)
	}
}

func TestBind_SwapWorksWithinOnePlayerAndOnTheGamepadColumn(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	f.Navigate(NavRight)
	onRow(t, f, "a")
	f.Navigate(NavConfirm)
	f.Bind(DevicePad, "b") // P1's b button
	if want := "b is Player 1 B: Enter swaps, Esc keeps"; f.OptionsPrompt() != want {
		t.Errorf("prompt = %q, want %q", f.OptionsPrompt(), want)
	}
	f.Navigate(NavConfirm)
	if p := f.Settings.Controls[0].Pad; p[0] != "b" || p[1] != "a" {
		t.Errorf("pad = %q, want a=b and b=a", p)
	}
}

func TestBind_CancellingAConflictKeepsBothBindings(t *testing.T) {
	f, rec := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	onRow(t, f, "a")
	gen := f.Settings.ControlsGen
	f.Navigate(NavConfirm)
	f.Bind(DeviceKeyboard, "J")
	navAll(f, NavUp, NavRight, NavPause) // only confirm and cancel answer
	if f.OptionsPrompt() == "" {
		t.Fatal("the question went away on an unrelated input")
	}
	f.Navigate(NavCancel)
	if f.Settings.Controls != NewSettings().Controls || f.Settings.ControlsGen != gen {
		t.Error("the cancel changed the bindings")
	}
	if f.OptionsPrompt() != "" || f.Screen != ScreenOptionsBindings {
		t.Errorf("prompt = %q screen = %v", f.OptionsPrompt(), f.Screen)
	}
	navAll(f, NavCancel, NavCancel)
	if len(rec.names) != 0 {
		t.Errorf("written %q, nothing changed", rec.names)
	}
}

func TestBind_TheKeyAlreadyOnTheRowChangesNothing(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	onRow(t, f, "a")
	gen := f.Settings.ControlsGen
	f.Navigate(NavConfirm)
	f.Bind(DeviceKeyboard, "F")
	if _, ok := f.WaitingForBinding(); ok || f.OptionsPrompt() != "" || f.Settings.ControlsGen != gen {
		t.Errorf("waiting or prompt %q or generation %d changed", f.OptionsPrompt(), f.Settings.ControlsGen)
	}
}

func TestBind_AConflictIsLookedForAmongTheSameKindOfDeviceOnly(t *testing.T) {
	f, _ := controlsFlow(t, map[string]string{"controls.json": `{"linux": {"p2": {"gamepad": {"start": "Q"}}}}`})
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavDown, NavConfirm)
	toPlayer(t, f, 0)
	onRow(t, f, "a")
	f.Navigate(NavConfirm)
	f.Bind(DeviceKeyboard, "Q") // P2's start button is named Q: another kind of device
	if f.OptionsPrompt() != "" || f.Settings.Controls[0].Keyboard[4] != "Q" {
		t.Errorf("prompt = %q keyboard = %q, want Q assigned", f.OptionsPrompt(), f.Settings.Controls[0].Keyboard)
	}
}

func TestBind_AButtonOfTheOtherPlayersPadIsNotAConflict(t *testing.T) {
	f, _ := controlsFlow(t, map[string]string{"controls.json": `{"linux": {"p2": {"gamepad": {"a": "guide"}}}}`})
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavDown, NavConfirm)
	toPlayer(t, f, 0)
	f.Navigate(NavRight)
	onRow(t, f, "a")
	f.Navigate(NavConfirm)
	f.Bind(DevicePad, "guide") // each player has a pad of their own
	if f.OptionsPrompt() != "" || f.Settings.Controls[0].Pad[0] != "guide" || f.Settings.Controls[1].Pad[0] != "guide" {
		t.Errorf("prompt = %q pads = %q / %q", f.OptionsPrompt(), f.Settings.Controls[0].Pad, f.Settings.Controls[1].Pad)
	}
}

func TestBindingsScreen_TheSameButtonOnBothPlayersPadsIsNotFlagged(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 0)
	for i, red := range f.OptionsFlagged() {
		if red {
			t.Errorf("default row %d is flagged", i)
		}
	}
}

func TestControlsMenu_ResetToDefaultsRestoresBothPlayersAndIsSavedOnLeaving(t *testing.T) {
	f, rec := controlsFlow(t, map[string]string{"controls.json": `{"linux": {
		"p1": {"keyboard": {"a": "U"}}, "p2": {"gamepad": {"start": "back"}}}}`})
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavDown, NavConfirm)
	gen := f.Settings.ControlsGen
	navAll(f, NavDown, NavDown, NavConfirm) // Reset to defaults
	if f.Settings.Controls != NewSettings().Controls || f.Settings.ControlsGen != gen+1 {
		t.Errorf("controls = %+v gen = %d", f.Settings.Controls, f.Settings.ControlsGen)
	}
	if f.Screen != ScreenOptionsControls {
		t.Errorf("screen = %v, want to stay on the controls menu", f.Screen)
	}
	navAll(f, NavConfirm) // Reset again: nothing changes
	if f.Settings.ControlsGen != gen+1 {
		t.Errorf("generation = %d after a useless reset", f.Settings.ControlsGen)
	}
	navAll(f, NavCancel)
	kb := decodeJSON(t, rec.files["config/controls.json"])["linux"].(map[string]any)["p1"].(map[string]any)["keyboard"].(map[string]any)
	if kb["a"] != "F" {
		t.Errorf("saved p1 a = %v, want F", kb["a"])
	}
}

func TestControls_SaveKeepsOtherPlatformsAndUnknownTopLevelKeys(t *testing.T) {
	f, rec := controlsFlow(t, map[string]string{"controls.json": `{"note": "mine",
		"windows": {"p1": {"keyboard": {"a": "Q"}}}, "linux": {"p1": {"keyboard": {"b": "B"}}}}`})
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavDown, NavConfirm)
	toPlayer(t, f, 0)
	onRow(t, f, "a")
	f.Navigate(NavConfirm)
	f.Bind(DeviceKeyboard, "U")
	navAll(f, NavCancel, NavCancel)
	got := decodeJSON(t, rec.files["config/controls.json"])
	if got["note"] != "mine" {
		t.Errorf("unknown key lost: %v", got)
	}
	win := got["windows"].(map[string]any)["p1"].(map[string]any)["keyboard"].(map[string]any)
	if win["a"] != "Q" {
		t.Errorf("windows section = %v", got["windows"])
	}
	kb := got["linux"].(map[string]any)["p1"].(map[string]any)["keyboard"].(map[string]any)
	if kb["a"] != "U" || kb["b"] != "B" {
		t.Errorf("linux p1 keyboard = %v, want a=U b=B", kb)
	}
}

func TestSaveOpenConcern_QuitOnTheBindingsScreenSavesTheControls(t *testing.T) {
	f, rec := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 1)
	onRow(t, f, "z")
	f.Navigate(NavConfirm)
	f.Bind(DeviceKeyboard, "U")
	f.SaveOpenConcern()
	if len(rec.names) != 1 || rec.names[0] != "config/controls.json" {
		t.Fatalf("written = %q, want controls.json", rec.names)
	}
	kb := decodeJSON(t, rec.files["config/controls.json"])["linux"].(map[string]any)["p2"].(map[string]any)["keyboard"].(map[string]any)
	if kb["z"] != "U" {
		t.Errorf("saved p2 keyboard = %v", kb)
	}
	f.SaveOpenConcern()
	if len(rec.names) != 1 {
		t.Errorf("written again: %q", rec.names)
	}
}

func TestBindingsScreen_BackReturnsToTheControlsMenuOnThatPlayer(t *testing.T) {
	f, _ := optionsFlow(t)
	toControls(t, f)
	toPlayer(t, f, 1)
	for range len(bindRows) {
		f.Navigate(NavDown)
	}
	f.Navigate(NavConfirm) // Back row
	if f.Screen != ScreenOptionsControls || f.OptionsCursor() != 1 {
		t.Errorf("screen = %v cursor = %d, want the controls menu on Player 2", f.Screen, f.OptionsCursor())
	}
}
