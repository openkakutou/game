package ui_test

import (
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/ui"
)

var (
	orange  = ui.Color{R: 1, G: 170.0 / 255, B: 0, A: 1}
	skyBlue = ui.Color{R: 0, G: 170.0 / 255, B: 1, A: 1}
	yellow  = ui.Color{R: 1, G: 1, B: 0, A: 1}
	magenta = ui.Color{R: 1, G: 0, B: 1, A: 1}
	warning = ui.Color{R: 220.0 / 255, G: 60.0 / 255, B: 60.0 / 255, A: 1}
)

func optionsLayout(t *testing.T, parts string) *ui.Layout {
	t.Helper()
	return layoutOf(t, screenGame(t, "options", parts, nil))
}

func drawOptions(l *ui.Layout, s ui.OptionsState) *ui.DrawList {
	var list ui.DrawList
	l.Options(&list, view720, s)
	return &list
}

// baseline of the single text op of s, or fails.
func only(t *testing.T, l *ui.DrawList, s string) ui.Op {
	t.Helper()
	ops := textOps(l, s)
	if len(ops) != 1 {
		t.Fatalf("text %q drawn %d times in %v", s, len(ops), onlyTexts(l))
	}
	return ops[0]
}

const valuedMenu = `"menu": { "anchor": "topLeft", "pivot": "topLeft", "spacing": 6, "style": "left", "activeStyle": "left",
	"valueStyle": "value", "valueOffset": [40, 0] }`

// controlsMenu is valuedMenu whose active style is the right-aligned yellow
// one, so the highlighted column can be told from the others.
const controlsMenu = `"menu": { "anchor": "topLeft", "pivot": "topLeft", "spacing": 6, "style": "left", "activeStyle": "hot",
	"valueStyle": "value", "valueOffset": [40, 0] }, "controls": { "slotStyle": "slot", "waitingStyle": "wait" }`

func TestOptions_AValueIsDrawnInTheValueStyleAtTheValueOffsetFromTheLabel(t *testing.T) {
	l := optionsLayout(t, valuedMenu)
	list := drawOptions(l, ui.OptionsState{Rows: []ui.Row{{Label: "Rounds", Values: [2]string{"3"}}, {Label: "Back"}}, Column: -1})
	label, value := only(t, list, "Rounds"), only(t, list, "3")
	// A left-aligned label starts at the box's left (0); the value's right
	// edge is 40 units = 288 px further, on the same line.
	near(t, "label x", label.X, 0)
	near(t, "value x", value.X, 288-newMeasurer(t, 36).width("3"))
	near(t, "same line", value.Y, label.Y)
	if value.Color != orange {
		t.Errorf("value color = %+v, want the value style's orange", value.Color)
	}
}

func TestOptions_BindingsPutTheKeyAndTheButtonInTwoColumnsAndHighlightTheChosenOne(t *testing.T) {
	l := optionsLayout(t, controlsMenu)
	rows := []ui.Row{{Label: "A", Values: [2]string{"F", "a"}}, {Label: "Back"}}
	m := newMeasurer(t, 36)
	list := drawOptions(l, ui.OptionsState{Rows: rows, Controls: true, Column: 0})
	// The key's right edge is 0.75 x 288 = 216 px, the button's 1.25 x 288 = 360 px.
	key, button := only(t, list, "F"), only(t, list, "a")
	near(t, "key x", key.X, 216-m.width("F"))
	near(t, "button x", button.X, 360-m.width("a"))
	if key.Color != yellow || button.Color != skyBlue {
		t.Errorf("colors key %+v button %+v, want the active style's yellow and the slot style's blue", key.Color, button.Color)
	}
	list = drawOptions(l, ui.OptionsState{Rows: rows, Controls: true, Column: 1})
	if only(t, list, "F").Color != skyBlue || only(t, list, "a").Color != yellow {
		t.Errorf("the pad column is not the highlighted one")
	}
}

func TestOptions_WaitingForAKeyShowsDotsInTheWaitingStyleInPlaceOfTheSlot(t *testing.T) {
	l := optionsLayout(t, controlsMenu)
	rows := []ui.Row{{Label: "A", Values: [2]string{"F", "a"}}, {Label: "B", Values: [2]string{"G", "b"}}}
	list := drawOptions(l, ui.OptionsState{Rows: rows, Controls: true, Column: 0, Waiting: true, Cursor: 0})
	if got := only(t, list, "..."); got.Color != magenta {
		t.Errorf("waiting color = %+v, want the waiting style's magenta", got.Color)
	}
	if len(textOps(list, "F")) != 0 || len(textOps(list, "G")) != 1 {
		t.Errorf("texts = %v, want the awaited slot (F) replaced and the other row's kept", onlyTexts(list))
	}
}

func TestOptions_AFlaggedRowIsDrawnInTheWarningColor(t *testing.T) {
	l := optionsLayout(t, controlsMenu)
	rows := []ui.Row{{Label: "A", Values: [2]string{"F", "a"}, Flagged: true}, {Label: "B", Values: [2]string{"G", "b"}}}
	list := drawOptions(l, ui.OptionsState{Rows: rows, Controls: true, Column: 0, Cursor: 1})
	for _, s := range []string{"A", "F", "a"} {
		if got := only(t, list, s).Color; got != warning {
			t.Errorf("%q color = %+v, want the warning red", s, got)
		}
	}
	if only(t, list, "B").Color == warning {
		t.Errorf("an unflagged row was drawn in the warning color")
	}
}

func TestOptions_TheTitleIsAPartAndThePromptAndNoticesStackUpFromTheHint(t *testing.T) {
	parts := valuedMenu + `, "title": { "style": "center", "anchor": "top", "offset": [0, 4] },
	  "hint": { "style": "center", "anchor": "bottom", "offset": [0, -5] }`
	l := optionsLayout(t, parts)
	list := drawOptions(l, ui.OptionsState{Title: "Audio", Prompt: "Press a key...", Notices: []string{"N1", "N2"}, Column: -1})
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128 // baseline below the line's center
	title := only(t, list, "Audio")
	near(t, "title x", title.X, (1280-m.width("Audio"))/2)
	near(t, "title baseline", title.Y, 28.8+18+shift) // top at 4 units, 36 px line
	// Hint line: bottom at 720-36 = 684, centered on 666; a notice is 1.5 lines (54 px) higher each.
	near(t, "prompt baseline", only(t, list, "Press a key...").Y, 666+shift)
	near(t, "first notice", only(t, list, "N1").Y, 612+shift)
	near(t, "second notice", only(t, list, "N2").Y, 558+shift)
	if only(t, list, "N1").Color != warning {
		t.Errorf("a notice is not in the warning color")
	}
}

func TestOptions_TheBuiltInScreenDrawsTitleRowsValueAndCursor(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{})
	l := ui.NewLayout(pack)
	rows := []ui.Row{{Label: "Fullscreen", Values: [2]string{"Off"}}, {Label: "Back"}}
	var list ui.DrawList
	l.Options(&list, view720, ui.OptionsState{Title: "Video", Rows: rows, Column: -1})
	got := shownTexts(&list)
	want := []string{"VIDEO", "Fullscreen", "Off", "Back"}
	if len(got) != len(want) {
		t.Fatalf("texts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("texts = %v, want %v", got, want)
			break
		}
	}
	if len(imageOps(&list, "images/background.png")) != 1 {
		t.Errorf("the background is missing")
	}
}

func TestPause_DimsTheFrozenMatchThenTitleAndMenu(t *testing.T) {
	parts := `"dim": { "color": "#0e0f16b3" },
	  "title": { "style": "center", "anchor": "center", "offset": [0, -22] },
	  "menu": { "anchor": "center", "pivot": "top", "spacing": 6, "style": "left", "activeStyle": "left" }`
	l := layoutOf(t, screenGame(t, "pause", parts, nil))
	var list ui.DrawList
	l.Pause(&list, view720, ui.PauseState{Title: "PAUSE", Entries: []string{"Continue", "Exit"}, ElapsedMs: 5000})
	dim := list.Ops[0]
	want := ui.Color{R: 14.0 / 255, G: 15.0 / 255, B: 22.0 / 255, A: 179.0 / 255}
	if dim.Kind != ui.OpRect || dim.Color != want || dim.X != 0 || dim.Y != 0 || dim.W != 1280 || dim.H != 720 {
		t.Errorf("first op = %+v, want the full-screen dim %+v", dim, want)
	}
	got := onlyTexts(&list)
	if len(got) != 3 || got[0] != "PAUSE" || got[1] != "Continue" || got[2] != "Exit" {
		t.Errorf("texts = %v, want PAUSE, Continue, Exit", got)
	}
	// Title: anchored at the center, 22 units (158.4 px) up: line centered on 201.6.
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	near(t, "title baseline", only(t, &list, "PAUSE").Y, 201.6+(float32(met.Ascent)-float32(met.Descent))/128)
}

func TestMatchSetup_ListsItsRowsWithTheirValuesUnderTheTitle(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{})
	l := ui.NewLayout(pack)
	rows := []ui.Row{{Label: "Rounds", Values: [2]string{"best of 3"}}, {Label: "Start match"}}
	var list ui.DrawList
	l.MatchSetup(&list, view720, ui.MatchSetupState{Title: "Match setup", Rows: rows, Cursor: 1})
	got := shownTexts(&list)
	want := []string{"MATCH SETUP", "Rounds", "best of 3", "Start match"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("texts = %v, want %v", got, want)
		}
	}
}

func TestVersus_PlacesBothNamesAndTheStageAndTheForegroundVS(t *testing.T) {
	parts := `"foreground": [ { "type": "text", "text": "VS", "style": "center", "anchor": "center" } ],
	  "name": { "p1": { "style": "left", "anchor": "bottomLeft", "offset": [8, -10] },
	            "p2": { "style": "right", "anchor": "bottomRight", "offset": [-8, -10] } },
	  "stage": { "style": "center", "anchor": "bottom", "offset": [0, -4] }`
	l := layoutOf(t, screenGame(t, "versus", parts, nil))
	var list ui.DrawList
	l.Versus(&list, view720, ui.VersusState{P1: "Alice", P2: "Bob", Stage: "Arena"})
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	// P1: bottom-left corner 57.6 px in and 72 px up: line 612..648, center 630.
	p1 := only(t, &list, "Alice")
	near(t, "p1 x", p1.X, 57.6)
	near(t, "p1 baseline", p1.Y, 630+shift)
	// P2: bottom-right corner at 1280-57.6.
	near(t, "p2 x", only(t, &list, "Bob").X, 1222.4-m.width("Bob"))
	// Stage: bottom center, 28.8 px up: line 655.2..691.2, center 673.2.
	stage := only(t, &list, "Arena")
	near(t, "stage x", stage.X, (1280-m.width("Arena"))/2)
	near(t, "stage baseline", stage.Y, 673.2+shift)
	vs := only(t, &list, "VS")
	near(t, "vs baseline", vs.Y, 360+shift)
	if list.Ops[len(list.Ops)-1].Kind != ui.OpText {
		t.Errorf("the foreground must be drawn last")
	}
}

func TestResults_ShowsTheWinnerTheMessageTheScoreAndTheMenuWithItsCursor(t *testing.T) {
	parts := `"winner": { "portrait": { "anchor": "left", "size": [null, 60] },
	                    "name": { "style": "left", "anchor": "left", "offset": [10, 34] } },
	  "message": { "style": "center", "anchor": "top", "offset": [20, 14] },
	  "score": { "style": "center", "anchor": "top", "offset": [20, 24] },
	  "menu": { "anchor": "center", "pivot": "top", "spacing": 6, "style": "left", "activeStyle": "left",
	            "cursor": { "type": "rect", "size": [20, 6], "color": "#ff0000" } }`
	l := layoutOf(t, screenGame(t, "results", parts, nil))
	var list ui.DrawList
	l.Results(&list, view720, ui.ResultsState{Message: "Player 1 wins", Score: "2 - 0", Winner: "Alice",
		Entries: []string{"Rematch", "Back to selection"}, Cursor: 1})
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	// Message: anchor top at 640 + 144 = 784, top at 100.8: a centered box.
	msg := only(t, &list, "Player 1 wins")
	near(t, "message x", msg.X, 784-m.width("Player 1 wins")/2)
	near(t, "message baseline", msg.Y, 118.8+shift)
	// Score: 24 units = 172.8 px down.
	near(t, "score baseline", only(t, &list, "2 - 0").Y, 190.8+shift)
	// Winner name: left edge 72 px in, 34 units = 244.8 px below the middle: center 604.8.
	name := only(t, &list, "Alice")
	near(t, "name x", name.X, 72)
	near(t, "name baseline", name.Y, 604.8+shift)
	// The menu hangs from the center (360): row 1 top at 360 + 43.2.
	cursor := rectOps(&list, red)
	near(t, "cursor y", cursor[0].Y, 403.2)
}

func TestScreens_AFrameAllocatesNothingOnceWarm(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{})
	l := ui.NewLayout(pack)
	rows := []ui.Row{{Label: "A", Values: [2]string{"F", "a"}, Flagged: true}, {Label: "Back"}}
	entries := []string{"Rematch", "Back"} // built once: a literal in the closure would be heap allocated by the test itself
	frames := map[string]func(*ui.DrawList){
		"options": func(d *ui.DrawList) {
			l.Options(d, view720, ui.OptionsState{Title: "Controls", Rows: rows, Controls: true, Waiting: true, Prompt: "Press a key...", Notices: []string{"n"}})
		},
		"pause": func(d *ui.DrawList) {
			l.Pause(d, view720, ui.PauseState{Title: "PAUSE", Entries: []string{"Continue", "Exit"}})
		},
		"setup": func(d *ui.DrawList) {
			l.MatchSetup(d, view720, ui.MatchSetupState{Title: "Match setup", Rows: rows})
		},
		"versus": func(d *ui.DrawList) {
			l.Versus(d, view720, ui.VersusState{P1: "Alice", P2: "Bob", Stage: "Arena"})
		},
		"results": func(d *ui.DrawList) {
			l.Results(d, view720, ui.ResultsState{Message: "m", Score: "s", Winner: "w", Entries: entries})
		},
	}
	for name, frame := range frames {
		var list ui.DrawList
		frame(&list)
		allocs := testing.AllocsPerRun(50, func() {
			list.Reset()
			frame(&list)
		})
		if allocs != 0 {
			t.Errorf("%s: %v allocations per frame, want 0", name, allocs)
		}
	}
}
