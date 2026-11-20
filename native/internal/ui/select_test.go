package ui_test

import (
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/ui"
)

// portraits is a PortraitSizes with a 50x25 small portrait and a 60x90 large
// one for every character but missing.
type portraits struct{ missing int }

func (p portraits) PortraitSize(character int, large bool) (w, h int, ok bool) {
	if character == p.missing {
		return 0, 0, false
	}
	if large {
		return 60, 90, true
	}
	return 50, 25, true
}

func roster(n int) []game.Entry {
	out := make([]game.Entry, n)
	for i := range out {
		out[i] = game.Entry{Name: string(rune('a' + i)), Display: "Fighter " + string(rune('A'+i))}
	}
	return out
}

// portraitOps are the portrait quads, in paint order.
func portraitOps(l *ui.DrawList, large bool) []ui.Op {
	var out []ui.Op
	for _, op := range l.Ops {
		if op.Kind == ui.OpPortrait && op.Portrait.Large == large {
			out = append(out, op)
		}
	}
	return out
}

const gridPart = `"grid": { "anchor": "topLeft", "pivot": "topLeft", "columns": 3, "rows": 2,
	"cellSize": [10, 10], "spacing": [2, 2],
	"cell": { "type": "rect", "color": "#ff0000" }, "emptyCell": { "type": "rect", "color": "#0000ff" } }`

func selectLayout(t *testing.T, parts string) *ui.Layout {
	t.Helper()
	return layoutOf(t, screenGame(t, "characterSelect", parts, nil))
}

func drawSelect(l *ui.Layout, s ui.SelectState) *ui.DrawList {
	var list ui.DrawList
	l.CharacterSelect(&list, view720, s)
	return &list
}

func TestCharacterSelect_TheGridPlacesACellOrAnEmptyCellInEverySlot(t *testing.T) {
	// 1280x720: a cell is 72 px, the spacing 14.4 px, so slot (column, row)
	// starts at (86.4 x column, 86.4 x row). 4 characters fill 4 of the 6 slots.
	list := drawSelect(selectLayout(t, gridPart), ui.SelectState{Roster: roster(4), Portraits: portraits{missing: -1}})
	cells := rectOps(list, red)
	wantX := []float32{0, 86.4, 172.8, 0}
	wantY := []float32{0, 0, 0, 86.4}
	if len(cells) != 4 {
		t.Fatalf("cells = %+v, want 4", cells)
	}
	for i, c := range cells {
		near(t, "cell x", c.X, wantX[i])
		near(t, "cell y", c.Y, wantY[i])
		near(t, "cell w", c.W, 72)
	}
	empty := rectOps(list, ui.Color{B: 1, A: 1})
	if len(empty) != 2 {
		t.Fatalf("empty cells = %+v, want 2", empty)
	}
	near(t, "empty x", empty[0].X, 86.4)
	near(t, "empty y", empty[0].Y, 86.4)
}

func TestCharacterSelect_TheGridScrollsToKeepTheFocusedCellInView(t *testing.T) {
	// 9 characters in rows of 3 = 3 rows, 2 shown: the window starts at the
	// focused row minus half the shown rows, within the grid.
	l := selectLayout(t, gridPart)
	cases := []struct {
		focus int
		first int
	}{{0, 0}, {4, 0}, {6, 3}, {8, 3}}
	for _, c := range cases {
		list := drawSelect(l, ui.SelectState{Roster: roster(9), Focus: c.focus, Portraits: portraits{missing: -1}})
		var got []int
		for _, op := range portraitOps(list, false) {
			got = append(got, op.Portrait.Character)
		}
		want := []int{c.first, c.first + 1, c.first + 2, c.first + 3, c.first + 4, c.first + 5}
		if len(got) != 6 {
			t.Fatalf("focus %d: portraits %v", c.focus, got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("focus %d shows characters %v, want %v", c.focus, got, want)
				break
			}
		}
	}
}

func TestCharacterSelect_ASmallPortraitIsFittedAndCenteredInItsCell(t *testing.T) {
	// A 50x25 portrait in a 72x72 cell: scale min(72/50, 72/25) = 1.44, so
	// 72x36, centered vertically: y = (72-36)/2.
	list := drawSelect(selectLayout(t, gridPart), ui.SelectState{Roster: roster(1), Portraits: portraits{missing: -1}})
	p := portraitOps(list, false)[0]
	near(t, "x", p.X, 0)
	near(t, "y", p.Y, 18)
	near(t, "w", p.W, 72)
	near(t, "h", p.H, 36)
	near(t, "u1", p.U1, 1)
}

func TestCharacterSelect_ACharacterWithoutPortraitShowsItsCellAlone(t *testing.T) {
	list := drawSelect(selectLayout(t, gridPart), ui.SelectState{Roster: roster(3), Portraits: portraits{missing: 1}})
	if len(rectOps(list, red)) != 3 {
		t.Errorf("a cell is missing")
	}
	got := portraitOps(list, false)
	if len(got) != 2 || got[0].Portrait.Character != 0 || got[1].Portrait.Character != 2 {
		t.Errorf("portraits = %+v, want characters 0 and 2", got)
	}
}

const cursorParts = gridPart + `, "cursor": {
	"p1": { "type": "rect", "color": "#00ff00", "size": [12, 12] },
	"p2": { "type": "rect", "color": "#ffff00", "size": [12, 12] },
	"both": { "type": "rect", "color": "#ff00ff" } }`

func TestCharacterSelect_EachShownSideHasACursorOnItsCell(t *testing.T) {
	l := selectLayout(t, cursorParts)
	s := ui.SelectState{Roster: roster(4), Portraits: portraits{missing: -1}}
	s.Sides = [2]ui.SideSelect{{Shown: true, Cursor: 1}, {Shown: true, Cursor: 3}}
	list := drawSelect(l, s)
	// A 12-unit (86.4 px) cursor centered in a 72 px cell is 7.2 px out on each side.
	p1 := rectOps(list, green)
	near(t, "p1 x", p1[0].X, 86.4-7.2)
	near(t, "p1 y", p1[0].Y, -7.2)
	p2 := rectOps(list, ui.Color{R: 1, G: 1, A: 1})
	near(t, "p2 x", p2[0].X, -7.2)
	near(t, "p2 y", p2[0].Y, 86.4-7.2)
	if len(rectOps(list, ui.Color{R: 1, B: 1, A: 1})) != 0 {
		t.Errorf("the shared-cell cursor was drawn for two separate cells")
	}
}

func TestCharacterSelect_BothSidesOnOneCellShowTheBothElementInstead(t *testing.T) {
	s := ui.SelectState{Roster: roster(4), Portraits: portraits{missing: -1}}
	s.Sides = [2]ui.SideSelect{{Shown: true, Cursor: 1}, {Shown: true, Cursor: 1}}
	list := drawSelect(selectLayout(t, cursorParts), s)
	both := rectOps(list, ui.Color{R: 1, B: 1, A: 1})
	if len(both) != 1 || len(rectOps(list, green)) != 0 {
		t.Fatalf("both = %+v, p1 cursors = %d: want only the both element", both, len(rectOps(list, green)))
	}
	near(t, "x", both[0].X, 86.4) // no size: it fills the cell
	near(t, "w", both[0].W, 72)
}

func TestCharacterSelect_ASideNotYetPickingHasNoCursorNorPortrait(t *testing.T) {
	s := ui.SelectState{Roster: roster(4), Portraits: portraits{missing: -1}}
	s.Sides = [2]ui.SideSelect{{Shown: true, Cursor: 0}}
	list := drawSelect(selectLayout(t, cursorParts+`, "portrait": { "p1": { "size": [null, 45] }, "p2": { "size": [null, 45] } }`), s)
	if len(rectOps(list, ui.Color{R: 1, G: 1, A: 1})) != 0 {
		t.Errorf("the idle side's cursor was drawn")
	}
	if len(portraitOps(list, true)) != 1 {
		t.Errorf("large portraits = %d, want only the shown side's", len(portraitOps(list, true)))
	}
}

const bigParts = `"portrait": { "p1": { "anchor": "left", "offset": [6, -10], "size": [null, 45] },
                "p2": { "anchor": "right", "offset": [-6, -10], "size": [null, 45] }, "mirrorP2": true },
  "name": { "p1": { "style": "left", "anchor": "left", "offset": [6, 16] },
            "p2": { "style": "right", "anchor": "right", "offset": [-6, 16] } }`

func TestCharacterSelect_TheHoveredCharactersLargePortraitAndNameAreShownPerSide(t *testing.T) {
	s := ui.SelectState{Roster: roster(4), Portraits: portraits{missing: -1}}
	s.Sides = [2]ui.SideSelect{{Shown: true, Cursor: 2}, {Shown: true, Cursor: 3}}
	list := drawSelect(selectLayout(t, bigParts), s)
	big := portraitOps(list, true)
	if len(big) != 2 || big[0].Portrait.Character != 2 || big[1].Portrait.Character != 3 {
		t.Fatalf("large portraits = %+v, want characters 2 and 3", big)
	}
	// Height 45 units = 324 px, a 60x90 image so 216 wide. P1: 6 units (43.2
	// px) from the left, centered 10 units (72 px) above the middle: top 360-72-162.
	near(t, "p1 x", big[0].X, 43.2)
	near(t, "p1 y", big[0].Y, 126)
	near(t, "p1 w", big[0].W, 216)
	near(t, "p1 u0", big[0].U0, 0)
	near(t, "p1 u1", big[0].U1, 1)
	// P2 hangs from the right edge (1280-43.2) and is mirrored.
	near(t, "p2 x", big[1].X, 1236.8-216)
	near(t, "p2 u0", big[1].U0, 1)
	near(t, "p2 u1", big[1].U1, 0)
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	name := only(t, list, "Fighter C")
	near(t, "name x", name.X, 43.2)
	near(t, "name baseline", name.Y, 475.2+shift) // 16 units (115.2 px) below the middle
	near(t, "p2 name x", only(t, list, "Fighter D").X, 1236.8-m.width("Fighter D"))
}

func TestCharacterSelect_MirrorP2FalseKeepsTheSecondPortraitUpright(t *testing.T) {
	parts := `"portrait": { "p1": { "size": [null, 45] }, "p2": { "size": [null, 45] }, "mirrorP2": false }`
	s := ui.SelectState{Roster: roster(2), Portraits: portraits{missing: -1}}
	s.Sides = [2]ui.SideSelect{{Shown: true}, {Shown: true, Cursor: 1}}
	big := portraitOps(drawSelect(selectLayout(t, parts), s), true)
	near(t, "p2 u0", big[1].U0, 0)
	near(t, "p2 u1", big[1].U1, 1)
}

func TestCharacterSelect_TheBuiltInScreenDrawsItsTitleGridAndCursor(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	l := ui.NewLayout(pack)
	var list ui.DrawList
	s := ui.SelectState{Title: "Player 1", Roster: roster(12), Portraits: portraits{missing: -1}}
	s.Sides[0] = ui.SideSelect{Shown: true, Cursor: 0}
	l.CharacterSelect(&list, view720, s)
	if got := shownTexts(&list); len(got) == 0 || got[0] != "PLAYER 1" {
		t.Errorf("texts = %v, want the title first", got)
	}
	if len(imageOps(&list, "images/cursor.png")) != 1 {
		t.Errorf("the animated cursor of side 1 is missing")
	}
	if len(portraitOps(&list, false)) != 12 {
		t.Errorf("small portraits = %d, want 12", len(portraitOps(&list, false)))
	}
}

func TestStageSelect_ShowsThePreviewPlaceholderTheNameAndBothArrows(t *testing.T) {
	parts := `"preview": { "anchor": "center", "offset": [0, -4], "size": [96, 54] },
	  "name": { "style": "center", "anchor": "bottom", "pivot": "top", "offset": [0, -22] },
	  "arrows": { "left": { "type": "rect", "color": "#ff0000", "anchor": "center", "offset": [-56, -4], "size": [6, 6] },
	              "right": { "type": "rect", "color": "#ff0000", "anchor": "center", "offset": [56, -4], "size": [6, 6] } }`
	l := layoutOf(t, screenGame(t, "stageSelect", parts, nil))
	var list ui.DrawList
	l.StageSelect(&list, view720, ui.StageSelectState{Title: "Stage", Roster: roster(3), Cursor: 1})
	// Preview: 96x54 units = 691.2x388.8 px centered on (640, 360-28.8).
	preview := rectOps(&list, ui.Color{R: 29.0 / 255, G: 28.0 / 255, B: 42.0 / 255, A: 1})
	if len(preview) != 1 {
		t.Fatalf("preview placeholders = %d, want 1", len(preview))
	}
	near(t, "preview x", preview[0].X, 294.4)
	near(t, "preview y", preview[0].Y, 136.8)
	near(t, "preview w", preview[0].W, 691.2)
	// Arrows: 6 units = 43.2 px, centered 403.2 px either side of 640.
	arrows := rectOps(&list, red)
	if len(arrows) != 2 {
		t.Fatalf("arrows = %d, want 2", len(arrows))
	}
	near(t, "left x", arrows[0].X, 640-403.2-21.6)
	near(t, "right x", arrows[1].X, 640+403.2-21.6)
	near(t, "arrow y", arrows[0].Y, 331.2-21.6)
	// Name: top at 100-22 = 78 units = 561.6 px from the top.
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	name := only(t, &list, "Fighter B")
	near(t, "name x", name.X, (1280-m.width("Fighter B"))/2)
	near(t, "name baseline", name.Y, 561.6+18+(float32(met.Ascent)-float32(met.Descent))/128)
}

func TestStageSelect_ALoneStageHasNoArrows(t *testing.T) {
	parts := `"arrows": { "left": { "type": "rect", "color": "#ff0000", "size": [6, 6] },
	              "right": { "type": "rect", "color": "#ff0000", "size": [6, 6] } }`
	l := layoutOf(t, screenGame(t, "stageSelect", parts, nil))
	var list ui.DrawList
	l.StageSelect(&list, view720, ui.StageSelectState{Roster: roster(1)})
	if len(rectOps(&list, red)) != 0 {
		t.Errorf("arrows drawn with nothing to step to")
	}
}

func TestVersus_DrawsBothLargePortraitsMirroringTheSecond(t *testing.T) {
	parts := `"portrait": { "p1": { "anchor": "left", "offset": [8, 0], "size": [null, 60] },
	                       "p2": { "anchor": "right", "offset": [-8, 0], "size": [null, 60] } }`
	l := layoutOf(t, screenGame(t, "versus", parts, nil))
	var list ui.DrawList
	l.Versus(&list, view720, ui.VersusState{P1: "a", P2: "b", P1Char: 4, P2Char: 5, Portraits: portraits{missing: -1}})
	big := portraitOps(&list, true)
	if len(big) != 2 || big[0].Portrait.Character != 4 || big[1].Portrait.Character != 5 {
		t.Fatalf("portraits = %+v", big)
	}
	// 60 units = 432 px high, 288 wide, centered vertically: top 144; 57.6 px in.
	near(t, "p1 x", big[0].X, 57.6)
	near(t, "p1 y", big[0].Y, 144)
	near(t, "p1 w", big[0].W, 288)
	near(t, "p2 x", big[1].X, 1222.4-288)
	near(t, "p2 u0", big[1].U0, 1) // mirrorP2 defaults to true
}

func TestResults_DrawsTheWinnersLargePortrait(t *testing.T) {
	parts := `"winner": { "portrait": { "anchor": "left", "offset": [10, 0], "size": [null, 60] },
	                     "name": { "style": "left", "anchor": "left", "offset": [10, 34] } }`
	l := layoutOf(t, screenGame(t, "results", parts, nil))
	var list ui.DrawList
	l.Results(&list, view720, ui.ResultsState{Winner: "w", WinnerChar: 7, Portraits: portraits{missing: -1}})
	big := portraitOps(&list, true)
	if len(big) != 1 || big[0].Portrait.Character != 7 {
		t.Fatalf("portraits = %+v, want the winner's", big)
	}
	near(t, "x", big[0].X, 72)
	near(t, "y", big[0].Y, 144)
	near(t, "u0", big[0].U0, 0)
}

func TestSelectScreens_AFrameAllocatesNothingOnceWarm(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	l := ui.NewLayout(pack)
	sel := ui.SelectState{Title: "Player 1", Roster: roster(25), Focus: 20, Portraits: portraits{missing: 3}}
	sel.Sides = [2]ui.SideSelect{{Shown: true, Cursor: 20}, {Shown: true, Cursor: 20}}
	stages := roster(5)
	frames := map[string]func(*ui.DrawList){
		"character select": func(d *ui.DrawList) { l.CharacterSelect(d, view720, sel) },
		"stage select": func(d *ui.DrawList) {
			l.StageSelect(d, view720, ui.StageSelectState{Title: "Stage", Roster: stages, Cursor: 2})
		},
		"versus": func(d *ui.DrawList) {
			l.Versus(d, view720, ui.VersusState{P1: "a", P2: "b", P1Char: 1, P2Char: 2, Portraits: portraits{missing: -1}})
		},
		"results": func(d *ui.DrawList) {
			l.Results(d, view720, ui.ResultsState{Winner: "w", WinnerChar: 1, Portraits: portraits{missing: -1}})
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

func TestPack_ExposesTheGridColumnsAndTheSortRuleOfTheSelectScreens(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	if got := pack.SelectColumns(); got != 10 {
		t.Errorf("built-in columns = %d, want 10", got)
	}
	if by, desc := pack.Sort("characterSelect"); by != "folder" || desc {
		t.Errorf("character sort = %q %v, want folder ascending", by, desc)
	}
	if by, desc := pack.Sort("stageSelect"); by != "displayName" || desc {
		t.Errorf("stage sort = %q %v, want displayName ascending", by, desc)
	}
}
