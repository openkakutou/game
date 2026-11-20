package ui_test

import (
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/ui"
)

// The list is 100 units wide (720 px at 720p) from the left edge, 10 units
// (72 px) down, 3 lines of 8 units (57.6 px); the text is 5 units (36 px)
// high; the track is 50 units (360 px) tall and the thumb 10 (72 px), both
// from 10 units down at the right edge.
const moveListParts = `
  "title": { "style": "default", "anchor": "top", "offset": [0, 2] },
  "list": { "style": "left", "anchor": "topLeft", "pivot": "topLeft", "offset": [0, 10],
            "size": [100, null], "visibleLines": 3, "lineSpacing": 8 },
  "buttons": { "sheet": "images/buttons.json", "frames": { "a": "a", "down": "down" } },
  "scrollbar": {
    "track": { "type": "rect", "anchor": "topRight", "pivot": "topRight", "offset": [0, 10], "size": [1, 50], "color": "#ff0000" },
    "thumb": { "type": "rect", "anchor": "topRight", "pivot": "topRight", "offset": [0, 10], "size": [1, 10], "color": "#00ff00" } }`

func moveListLayout(t *testing.T) *ui.Layout {
	t.Helper()
	extra := fstest.MapFS{
		"images/buttons.json": {Data: []byte(`{"image": "buttons.png", "frames": {"a": [0, 0, 32, 32], "down": [32, 0, 32, 32]}}`)},
		"images/buttons.png":  {Data: pngOf(t, 64, 32)},
	}
	return layoutOf(t, screenGame(t, "moveList", moveListParts, extra))
}

func run(text, icon string) game.MoveRun { return game.MoveRun{Text: text, Icon: icon} }

func cell(a game.MoveAlign, runs ...game.MoveRun) game.MoveCell {
	return game.MoveCell{Align: a, Runs: runs}
}

var sampleLines = []game.MoveLine{
	{cell(game.MoveLeft, run("Move", "")), cell(game.MoveRight, run("A", "a"), run("+", ""), run("F", "forward"))},
	{cell(game.MoveCenter, run("mid", "")), cell(game.MoveCenter, run("dle", ""))},
	{cell(game.MoveLeft, run("x", "")), cell(game.MoveLeft, run("y", ""))},
	{cell(game.MoveLeft, run("d", ""))},
	{cell(game.MoveLeft, run("e", ""))},
}

func drawMoveList(l *ui.Layout, s ui.MoveListState) *ui.DrawList {
	var list ui.DrawList
	s.ElapsedMs = 10000
	l.MoveList(&list, view720, s)
	return &list
}

func TestMoveList_CellsSitByTheirAlignmentWithIconsAsTallAsTheText(t *testing.T) {
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	list := drawMoveList(moveListLayout(t), ui.MoveListState{Title: "Alice", Lines: sampleLines})
	// Line 0 is centered at 72 + 28.8 px. Left cell at the left edge.
	move := only(t, list, "Move")
	near(t, "left cell x", move.X, 0)
	near(t, "left cell baseline", move.Y, 100.8+shift)
	// Right cell: an icon (36 px wide, the text's height), "+" and the "F"
	// the pack has no icon for, ending at the 720 px edge.
	right := 720 - (36 + m.width("+") + m.width("F"))
	icons := imageOps(list, "images/buttons.png")
	if len(icons) != 1 {
		t.Fatalf("%d icons drawn, want 1 (forward has no icon)", len(icons))
	}
	icon := icons[0]
	near(t, "icon x", icon.X, right)
	near(t, "icon y", icon.Y, 100.8-18)
	near(t, "icon w", icon.W, 36)
	near(t, "icon h", icon.H, 36)
	near(t, "icon u0", icon.U0, 0)
	near(t, "icon u1", icon.U1, 0.5) // frame a is the left half of the 64 px image
	near(t, "plus x", only(t, list, "+").X, right+36)
	near(t, "fallback text x", only(t, list, "F").X, right+36+m.width("+"))
	// Line 1: the first center cell is centered, the next one follows it.
	mid := only(t, list, "mid")
	near(t, "center cell x", mid.X, (720-m.width("mid"))/2)
	near(t, "next center cell x", only(t, list, "dle").X, (720+m.width("mid"))/2)
	near(t, "line 1 baseline", mid.Y, 158.4+shift)
	// Line 2: left cells one after the other.
	near(t, "second left cell x", only(t, list, "y").X, m.width("x"))
	if got := shownTexts(list); got[0] != "Alice" {
		t.Errorf("texts = %v, want the title first", got)
	}
}

func TestMoveList_ShowsOnlyTheWindowFromTheFirstLine(t *testing.T) {
	list := drawMoveList(moveListLayout(t), ui.MoveListState{Title: "Alice", Lines: sampleLines, Top: 2})
	got := shownTexts(list)
	want := []string{"Alice", "x", "y", "d", "e"}
	if len(got) != len(want) {
		t.Fatalf("texts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("texts = %v, want %v", got, want)
		}
	}
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	near(t, "first shown line baseline", only(t, list, "x").Y, 100.8+shift)
	near(t, "third shown line baseline", only(t, list, "e").Y, 72+2*57.6+28.8+shift)
}

func TestMoveList_ScrollbarThumbFollowsTheFirstLine(t *testing.T) {
	ten := make([]game.MoveLine, 10) // 7 lines can be the first of a 3 line window
	for _, c := range []struct {
		top  int
		y    float32
		what string
	}{
		{0, 72, "top"},
		{3, 72 + 288.0*3/7, "three lines down"}, // track 72..432, thumb 72 tall: 288 px of travel
		{7, 360, "last"},
	} {
		list := drawMoveList(moveListLayout(t), ui.MoveListState{Lines: ten, Top: c.top})
		track, thumb := rectOps(list, red), rectOps(list, green)
		if len(track) != 1 || len(thumb) != 1 {
			t.Fatalf("%s: %d track and %d thumb rects", c.what, len(track), len(thumb))
		}
		near(t, c.what+": track y", track[0].Y, 72)
		near(t, c.what+": track h", track[0].H, 360)
		near(t, c.what+": thumb y", thumb[0].Y, c.y)
		near(t, c.what+": thumb h", thumb[0].H, 72)
		near(t, c.what+": thumb x", thumb[0].X, 1280-7.2)
	}
}

func TestMoveList_NoScrollbarWhenTheListFitsTheWindow(t *testing.T) {
	list := drawMoveList(moveListLayout(t), ui.MoveListState{Lines: sampleLines[:3]})
	if n := len(rectOps(list, red)) + len(rectOps(list, green)); n != 0 {
		t.Errorf("%d scrollbar rects for a list of exactly the window", n)
	}
}

func TestMoveList_AFrameAllocatesNothingOnceWarm(t *testing.T) {
	l := moveListLayout(t)
	state := ui.MoveListState{Title: "Alice", Lines: sampleLines, Top: 1, ElapsedMs: 10000}
	var list ui.DrawList
	l.MoveList(&list, view720, state)
	allocs := testing.AllocsPerRun(50, func() {
		list.Reset()
		l.MoveList(&list, view720, state)
	})
	if allocs != 0 {
		t.Errorf("%v allocations per frame, want 0", allocs)
	}
}

func TestMoveList_BuiltInScreenpackDrawsIt(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	if got := pack.MoveListWindow(); got != 14 {
		t.Errorf("the default pack shows %d lines, want 14", got)
	}
	list := drawMoveList(ui.NewLayout(pack), ui.MoveListState{Title: "Kung Fu Man", Lines: sampleLines})
	if got := shownTexts(list)[0]; got != "KUNG FU MAN" { // the default pack's heading style is uppercase
		t.Errorf("first text = %q, want the title", got)
	}
}
