package ui_test

import (
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/ui"
)

// horizontalMenu hangs its first entry from the top left corner (10 units in
// and down) and puts the next ones 20 units to the right each.
const horizontalMenu = `"menu": { "anchor": "topLeft", "pivot": "topLeft", "offset": [10, 10], "direction": "horizontal", "spacing": 20,
	"style": "center", "activeStyle": "center", "cursor": { "type": "rect", "size": [20, 6], "color": "#ff0000" } }`

func TestContinue_AHorizontalMenuPlacesEntriesSpacingApartOnOneLine(t *testing.T) {
	l := layoutOf(t, screenGame(t, "continue", horizontalMenu, nil))
	var list ui.DrawList
	l.Continue(&list, view720, ui.ContinueState{Entries: []string{"Yes", "No"}, Cursor: 1})
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	widest := max(m.width("Yes"), m.width("No"))
	// The first cell's top left corner is at 72, 72 px (10 units), the cells are as wide as the
	// widest label and 36 px high (5 units), 144 px (20 units) apart; labels are centered in them.
	yes, no := only(t, &list, "Yes"), only(t, &list, "No")
	near(t, "Yes x", yes.X, 72+(widest-m.width("Yes"))/2)
	near(t, "No x", no.X, 216+(widest-m.width("No"))/2)
	near(t, "Yes baseline", yes.Y, 90+shift)
	near(t, "No baseline", no.Y, 90+shift)
	// The 20 x 6 unit cursor (144 x 43.2 px) is centered in the active cell.
	cursor := rectOps(&list, red)
	if len(cursor) != 1 {
		t.Fatalf("%d cursors drawn, want 1", len(cursor))
	}
	near(t, "cursor x", cursor[0].X, 216+widest/2-72)
	near(t, "cursor y", cursor[0].Y, 68.4)
	near(t, "cursor w", cursor[0].W, 144)
	near(t, "cursor h", cursor[0].H, 43.2)
}

func TestContinue_DrawsThePortraitMessageCountdownAndMenu(t *testing.T) {
	parts := `"portrait": { "anchor": "left", "offset": [10, 0], "size": [null, 60] },
	  "message": { "style": "center", "anchor": "top", "offset": [20, 14] },
	  "countdown": { "style": "center", "anchor": "center", "offset": [20, -4] },` + horizontalMenu
	l := layoutOf(t, screenGame(t, "continue", parts, nil))
	var list ui.DrawList
	l.Continue(&list, view720, ui.ContinueState{Message: "Continue?", Countdown: "10", Char: 3, Portraits: portraits{missing: -1},
		Entries: []string{"Yes", "No"}})
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	// 60 units high is 432 px, 288 wide for a 60 x 90 portrait; its left edge is 72 px in, centered vertically.
	var portrait []ui.Op
	for _, op := range list.Ops {
		if op.Kind == ui.OpPortrait {
			portrait = append(portrait, op)
		}
	}
	if len(portrait) != 1 || portrait[0].Portrait.Character != 3 {
		t.Fatalf("portrait ops = %+v, want one for character 3", portrait)
	}
	near(t, "portrait x", portrait[0].X, 72)
	near(t, "portrait y", portrait[0].Y, 144)
	near(t, "portrait w", portrait[0].W, 288)
	near(t, "portrait h", portrait[0].H, 432)
	// Message and countdown are centered on the point 20 units (144 px) right of the middle.
	near(t, "message x", only(t, &list, "Continue?").X, 784-m.width("Continue?")/2)
	near(t, "message baseline", only(t, &list, "Continue?").Y, 118.8+shift)
	near(t, "countdown x", only(t, &list, "10").X, 784-m.width("10")/2)
	near(t, "countdown baseline", only(t, &list, "10").Y, 331.2+shift)
}

func TestContinue_TheBuiltInScreenLaysYesAndNoSideBySide(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	l := ui.NewLayout(pack)
	var list ui.DrawList
	l.Continue(&list, view720, ui.ContinueState{Message: "Continue?", Countdown: "7", Entries: []string{"Yes", "No"}})
	yes, no := only(t, &list, "Yes"), only(t, &list, "No")
	if no.X <= yes.X || no.Y != yes.Y {
		t.Errorf("Yes at %v,%v and No at %v,%v, want No to the right on the same line", yes.X, yes.Y, no.X, no.Y)
	}
	if len(imageOps(&list, "images/background.png")) != 1 {
		t.Errorf("the background is missing")
	}
}

func TestVictory_PlacesThePortraitTheNameAndTheMessage(t *testing.T) {
	parts := `"portrait": { "anchor": "center", "offset": [0, -8], "size": [null, 60] },
	  "name": { "style": "center", "anchor": "bottom", "pivot": "bottom", "offset": [0, -18] },
	  "message": { "style": "center", "anchor": "bottom", "pivot": "bottom", "offset": [0, -8] }`
	l := layoutOf(t, screenGame(t, "victory", parts, nil))
	var list ui.DrawList
	l.Victory(&list, view720, ui.VictoryState{Name: "Alice", Message: "Congratulations!", Char: 2, Portraits: portraits{missing: -1}})
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	var portrait []ui.Op
	for _, op := range list.Ops {
		if op.Kind == ui.OpPortrait {
			portrait = append(portrait, op)
		}
	}
	if len(portrait) != 1 || portrait[0].Portrait.Character != 2 {
		t.Fatalf("portrait ops = %+v, want one for character 2", portrait)
	}
	// Centered on 640, 302.4 (8 units above the middle), 288 x 432 px.
	near(t, "portrait x", portrait[0].X, 496)
	near(t, "portrait y", portrait[0].Y, 86.4)
	// The name's bottom edge is 18 units (129.6 px) above the screen's bottom: 36 px line centered on 572.4.
	near(t, "name x", only(t, &list, "Alice").X, 640-m.width("Alice")/2)
	near(t, "name baseline", only(t, &list, "Alice").Y, 572.4+shift)
	// The message's bottom edge is 8 units (57.6 px) above: centered on 644.4.
	near(t, "message baseline", only(t, &list, "Congratulations!").Y, 644.4+shift)
}

func TestVictory_TheBuiltInScreenDrawsItsBackgroundAndTexts(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	l := ui.NewLayout(pack)
	var list ui.DrawList
	l.Victory(&list, view720, ui.VictoryState{Name: "Alice", Message: "Congratulations!", ElapsedMs: 5000})
	if len(imageOps(&list, "images/background.png")) != 1 {
		t.Errorf("the background is missing")
	}
	if len(textOps(&list, "ALICE")) == 0 || len(textOps(&list, "Congratulations!")) == 0 { // the name style is uppercase
		t.Errorf("texts = %v, want the name and the message", shownTexts(&list))
	}
}

func TestPack_ContinueAndVictoryScreensHaveTheirOwnSoundsAndMusic(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	for _, name := range []string{"tick", "gameOver"} {
		if a, ok := pack.Sound("continue", name); !ok || a.Path == "" {
			t.Errorf("continue sound %q: %+v, %v, want the file of the continue screen's sounds", name, a, ok)
		}
	}
	if _, ok := pack.Music("continue"); ok {
		t.Errorf("the continue screen has music, want silence (its music is null)")
	}
	if _, ok := pack.Music("victory"); !ok {
		t.Errorf("the victory screen has no music, want the pack's root music")
	}
}

func TestPack_VictoryDurationIsTheScreensOrEightSeconds(t *testing.T) {
	builtIn, _ := ui.LoadPack(fstest.MapFS{}, "default")
	if got := builtIn.VictoryDurationMs(); got != 8000 {
		t.Errorf("built-in duration = %d ms, want 8000", got)
	}
	pack, notices := ui.LoadPack(screenGame(t, "victory", `"durationMs": 2500`, nil), "default")
	if len(notices) > 0 {
		t.Fatal(notices)
	}
	if got := pack.VictoryDurationMs(); got != 2500 {
		t.Errorf("duration = %d ms, want 2500", got)
	}
}
