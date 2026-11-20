package ui_test

import (
	"math"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/ui"
)

func bgRect(layout string) string {
	return `"background": [{ "type": "rect", ` + layout + `, "color": "#ff0000" }]`
}

func TestTitle_AnchorBottomRightWithOffsetLandsOnLiteralPixels(t *testing.T) {
	// 1280x720: 1 unit = 7.2 px. Anchor (1280, 720) + offset (-14.4, -14.4);
	// size 10x5 units = 72x36 px; the pivot defaults to the anchor, so the
	// rect's bottom-right corner sits there: x = 1265.6-72, y = 705.6-36.
	list := draw(layoutOf(t, gameWith(t, bgRect(`"anchor": "bottomRight", "offset": [-2, -2], "size": [10, 5]`), nil)), menuState(nil, 0))
	r := rectOps(list, red)
	if len(r) != 1 {
		t.Fatalf("red rects = %+v", r)
	}
	near(t, "x", r[0].X, 1193.6)
	near(t, "y", r[0].Y, 669.6)
	near(t, "w", r[0].W, 72)
	near(t, "h", r[0].H, 36)
}

func TestTitle_PivotCenterOnTopLeftAnchorShiftsByHalfTheSize(t *testing.T) {
	// Anchor (0,0) + offset (72, 72); size 144x72 px; pivot center: x = 72-72, y = 72-36.
	list := draw(layoutOf(t, gameWith(t, bgRect(`"anchor": "topLeft", "pivot": "center", "offset": [10, 10], "size": [20, 10]`), nil)), menuState(nil, 0))
	r := rectOps(list, red)
	near(t, "x", r[0].X, 0)
	near(t, "y", r[0].Y, 36)
}

func TestTitle_ElementsAreCenteredByDefault(t *testing.T) {
	// Anchor (640, 360), size 144x72 px, pivot center: (568, 324).
	list := draw(layoutOf(t, gameWith(t, bgRect(`"size": [20, 10]`), nil)), menuState(nil, 0))
	r := rectOps(list, red)
	near(t, "x", r[0].X, 568)
	near(t, "y", r[0].Y, 324)
}

func logoPart(layout string) string {
	return `"logo": { "type": "image", "image": "images/logo.png", ` + layout + ` }`
}

func imagePack(t *testing.T, part string, w, h int) *ui.Layout {
	t.Helper()
	extra := fstest.MapFS{"images/logo.png": {Data: pngOf(t, w, h)}}
	return layoutOf(t, gameWith(t, part, extra))
}

func TestTitle_ASizeSideSetToNullKeepsTheImageAspect(t *testing.T) {
	// A 200x100 image (2:1). [50, null]: width 360 px, so height 180.
	list := draw(imagePack(t, logoPart(`"anchor": "topLeft", "size": [50, null]`), 200, 100), menuState(nil, 0))
	q := imageOps(list, "images/logo.png")[0]
	near(t, "w", q.W, 360)
	near(t, "h", q.H, 180)
	// [null, 50]: height 360 px, so width 720.
	list = draw(imagePack(t, logoPart(`"anchor": "topLeft", "size": [null, 50]`), 200, 100), menuState(nil, 0))
	q = imageOps(list, "images/logo.png")[0]
	near(t, "w", q.W, 720)
	near(t, "h", q.H, 360)
}

func TestTitle_AnImageWithoutSizeIsOneImagePixelPer240thOfTheHeight(t *testing.T) {
	// 720/240 = 3 px per image pixel: 120x60 becomes 360x180.
	list := draw(imagePack(t, logoPart(`"anchor": "topLeft"`), 120, 60), menuState(nil, 0))
	q := imageOps(list, "images/logo.png")[0]
	near(t, "w", q.W, 360)
	near(t, "h", q.H, 180)
	near(t, "u1", q.U1, 1)
	near(t, "v1", q.V1, 1)
}

func TestTitle_TileBothIsOneQuadRepeatingTheImageAtItsNaturalSize(t *testing.T) {
	// Quad 100x100 units = 720x720 px; one 64x64 image is 192x192 px, so the
	// texture coordinates run to 720/192 = 3.75 in both directions.
	part := `"background": [{ "type": "image", "image": "images/logo.png", "anchor": "topLeft", "size": [100, 100], "tile": "both" }]`
	list := draw(imagePack(t, part, 64, 64), menuState(nil, 0))
	q := imageOps(list, "images/logo.png")
	if len(q) != 1 {
		t.Fatalf("quads = %+v", q)
	}
	near(t, "w", q[0].W, 720)
	near(t, "u1", q[0].U1, 3.75)
	near(t, "v1", q[0].V1, 3.75)
}

func TestTitle_TileXRepeatsOnlyHorizontally(t *testing.T) {
	// Quad 100x50 units = 720x360 px; the cell is 192 wide, and as high as the quad.
	part := `"background": [{ "type": "image", "image": "images/logo.png", "anchor": "topLeft", "size": [100, 50], "tile": "x" }]`
	q := imageOps(draw(imagePack(t, part, 64, 64), menuState(nil, 0)), "images/logo.png")[0]
	near(t, "u1", q.U1, 3.75)
	near(t, "v1", q.V1, 1)
}

func TestTitle_OpacityScalesTheColorAlpha(t *testing.T) {
	list := draw(layoutOf(t, gameWith(t, `"background": [{ "type": "rect", "size": [10, 10], "color": "#ff0000", "opacity": 0.5 }]`, nil)), menuState(nil, 0))
	r := rectOps(list, ui.Color{R: 1, A: 0.5})
	if len(r) != 1 {
		t.Errorf("no half-transparent red rect: %+v", list.Ops)
	}
}

func TestTitle_InvisibleElementsAreNotDrawn(t *testing.T) {
	list := draw(layoutOf(t, gameWith(t, `"background": [{ "type": "rect", "size": [10, 10], "color": "#ff0000", "visible": false }]`, nil)), menuState(nil, 0))
	if len(rectOps(list, red)) != 0 {
		t.Errorf("an invisible rect was drawn")
	}
}

func TestTitle_ColorsReadRRGGBBAA(t *testing.T) {
	part := `"menu": { "anchor": "topLeft", "spacing": 5, "style": "left", "activeStyle": "left",
	          "cursor": { "type": "rect", "size": [10, 3], "color": "#ff5c3833" } }`
	list := draw(layoutOf(t, gameWith(t, part, nil)), menuState([]string{"a"}, 0))
	want := ui.Color{R: 1, G: 92.0 / 255, B: 56.0 / 255, A: 51.0 / 255}
	if len(rectOps(list, want)) != 1 {
		t.Errorf("cursor color not #ff5c3833: %+v", list.Ops)
	}
}

const scrollingMenu = `"menu": { "anchor": "topLeft", "pivot": "topLeft", "spacing": 5, "visibleEntries": 4,
	"style": "left", "activeStyle": "left", "cursor": { "type": "rect", "size": [20, 3], "color": "#ff0000" } }`

var tenEntries = []string{"item0", "item1", "item2", "item3", "item4", "item5", "item6", "item7", "item8", "item9"}

func TestTitle_MenuScrollsToKeepTheCursorVisible(t *testing.T) {
	l := layoutOf(t, gameWith(t, scrollingMenu, nil))
	cases := []struct {
		cursor int
		want   []string
	}{
		{0, []string{"item0", "item1", "item2", "item3"}},
		{7, []string{"item5", "item6", "item7", "item8"}}, // 7-4/2 = 5
		{9, []string{"item6", "item7", "item8", "item9"}}, // the last window
	}
	for _, c := range cases {
		got := onlyTexts(draw(l, menuState(tenEntries, c.cursor)))
		if len(got) != 4 || got[0] != c.want[0] || got[3] != c.want[3] {
			t.Errorf("cursor %d shows %v, want %v", c.cursor, got, c.want)
		}
	}
}

func TestTitle_MenuCursorIsDrawnBehindTheActiveEntryOnItsRow(t *testing.T) {
	list := draw(layoutOf(t, gameWith(t, scrollingMenu, nil)), menuState(tenEntries, 7))
	cursorAt, activeAt, previousAt := -1, -1, -1
	for i, op := range list.Ops {
		switch {
		case op.Kind == ui.OpRect:
			cursorAt = i
		case op.Kind == ui.OpText && textOf(list, op) == "item7":
			activeAt = i
		case op.Kind == ui.OpText && textOf(list, op) == "item6":
			previousAt = i
		}
	}
	if !(previousAt < cursorAt && cursorAt < activeAt) {
		t.Fatalf("order: item6 at %d, cursor at %d, item7 at %d", previousAt, cursorAt, activeAt)
	}
	// Window starts at item5, so item7 is row 2: row top = 2*5*7.2 = 72; the
	// 3-unit cursor (21.6 px) is centered in the 36 px row: 72 + 7.2.
	c := list.Ops[cursorAt]
	near(t, "cursor y", c.Y, 79.2)
	near(t, "cursor h", c.H, 21.6)
	near(t, "cursor w", c.W, 144)
}

func TestTitle_MenuEntriesAreAlignedInsideTheWidestEntry(t *testing.T) {
	m := newMeasurer(t, 36) // size 5 units at 720 px high
	wide, narrow := m.width("wide entry"), m.width("ab")
	cases := map[string]float32{"left": 0, "center": (wide - narrow) / 2, "right": wide - narrow}
	for style, want := range cases {
		part := `"menu": { "anchor": "topLeft", "pivot": "topLeft", "spacing": 5, "style": "` + style + `", "activeStyle": "` + style + `" }`
		list := draw(layoutOf(t, gameWith(t, part, nil)), menuState([]string{"ab", "wide entry"}, 1))
		got := textOps(list, "ab")
		if len(got) != 1 {
			t.Fatalf("%s: ab ops = %+v", style, got)
		}
		near(t, style+" x", got[0].X, want)
	}
}

func TestTitle_SingleTextIsPlacedByItsOwnBox(t *testing.T) {
	// Version text, size 5 units (36 px), anchored bottomRight with [-2,-2]:
	// the box's right edge is 1280-14.4 and its bottom 720-14.4, and the
	// baseline centers the font's ascent+descent in the 36 px line.
	part := `"version": { "style": "default", "anchor": "bottomRight", "offset": [-2, -2] }`
	st := menuState(nil, 0)
	st.Version = "1.2.3"
	op := textOps(draw(layoutOf(t, gameWith(t, part, nil)), st), "1.2.3")
	if len(op) != 1 {
		t.Fatalf("version ops = %+v", op)
	}
	m := newMeasurer(t, 36)
	met := m.face.Metrics()
	asc, desc := float32(met.Ascent)/64, float32(met.Descent)/64
	near(t, "x", op[0].X, 1265.6-m.width("1.2.3"))
	near(t, "baseline", op[0].Y, 687.6+(asc-desc)/2) // box 669.6..705.6, middle 687.6
	if op[0].Px != 36 {
		t.Errorf("px = %d, want 36", op[0].Px)
	}
}

func TestTitle_NoVersionTextWhenTheGameHasNoVersion(t *testing.T) {
	part := `"version": { "style": "default", "anchor": "bottomRight" }`
	if got := onlyTexts(draw(layoutOf(t, gameWith(t, part, nil)), menuState(nil, 0))); len(got) != 0 {
		t.Errorf("texts = %v, want none", got)
	}
}

const blinking = `"pressStart": { "style": "default", "text": "Press start", "blinkMs": 600 }`

func TestTitle_PressStartBlinksOnForBlinkMsThenOffForBlinkMs(t *testing.T) {
	l := layoutOf(t, gameWith(t, blinking, nil))
	cases := []struct {
		elapsed int
		visible bool
	}{{0, true}, {599, true}, {600, false}, {1199, false}, {1200, true}}
	for _, c := range cases {
		list := draw(l, ui.TitleState{Phase: ui.PhasePressStart, ElapsedMs: c.elapsed})
		if got := len(textOps(list, "Press start")) == 1; got != c.visible {
			t.Errorf("at %d ms visible = %v, want %v", c.elapsed, got, c.visible)
		}
	}
}

func TestTitle_PressStartWithBlinkMsZeroIsSteady(t *testing.T) {
	l := layoutOf(t, gameWith(t, `"pressStart": { "style": "default", "text": "Press start", "blinkMs": 0 }`, nil))
	list := draw(l, ui.TitleState{Phase: ui.PhasePressStart, ElapsedMs: 700})
	if len(textOps(list, "Press start")) != 1 {
		t.Errorf("a steady prompt was hidden")
	}
}

func TestTitle_PhasesShowTheirOwnPart(t *testing.T) {
	part := blinking + `, "menu": { "style": "default", "activeStyle": "default", "spacing": 5 }`
	l := layoutOf(t, gameWith(t, part, nil))
	press := onlyTexts(draw(l, ui.TitleState{Phase: ui.PhasePressStart}))
	if len(press) != 1 || press[0] != "Press start" {
		t.Errorf("press start phase texts = %v", press)
	}
	menu := onlyTexts(draw(l, menuState([]string{"Versus", "Options"}, 0)))
	if len(menu) != 2 || menu[0] != "Versus" || menu[1] != "Options" {
		t.Errorf("menu phase texts = %v", menu)
	}
}

func TestTitle_UppercaseStyleDrawsCapitals(t *testing.T) {
	l := layoutOf(t, gameWith(t, `"pressStart": { "style": "upper", "text": "Press start" }`, nil))
	got := onlyTexts(draw(l, ui.TitleState{Phase: ui.PhasePressStart}))
	if len(got) != 1 || got[0] != "PRESS START" {
		t.Errorf("texts = %v, want [PRESS START]", got)
	}
}

func TestTitle_OutlineAndShadowAreExtraRunsBehindTheFill(t *testing.T) {
	l := layoutOf(t, gameWith(t, `"pressStart": { "style": "fancy", "text": "Hi" }`, nil))
	list := draw(l, ui.TitleState{Phase: ui.PhasePressStart})
	var texts []ui.Op
	for _, op := range list.Ops {
		if op.Kind == ui.OpText {
			texts = append(texts, op)
		}
	}
	// One shadow, eight outline runs, then the fill on top.
	if len(texts) != 10 {
		t.Fatalf("text ops = %d, want 10", len(texts))
	}
	fill := texts[9]
	if fill.Color != green {
		t.Errorf("last run color = %+v, want the green fill", fill.Color)
	}
	shadow := texts[0]
	if shadow.Color != (ui.Color{A: 128.0 / 255}) {
		t.Errorf("first run color = %+v, want the shadow", shadow.Color)
	}
	near(t, "shadow dx", shadow.X-fill.X, 7.2) // offset 1 unit
	near(t, "shadow dy", shadow.Y-fill.Y, 7.2)
	for _, o := range texts[1:9] {
		if o.Color != (ui.Color{B: 1, A: 1}) {
			t.Errorf("outline color = %+v, want blue", o.Color)
		}
		dx, dy := float64(o.X-fill.X), float64(o.Y-fill.Y)
		if math.Max(math.Abs(dx), math.Abs(dy)) < 3.59 || math.Max(math.Abs(dx), math.Abs(dy)) > 3.61 { // width 0.5 unit = 3.6 px
			t.Errorf("outline offset (%v, %v) is not 3.6 px out", dx, dy)
		}
	}
}

const logoScale = `"logo": { "type": "image", "image": "images/logo.png", "anchor": "topLeft", "size": [50, null],
	"transition": { "in": { "kind": "scale", "from": 0.5, "ms": 400, "easing": "linear" } } }`

func TestTitle_LogoScalesInAroundItsCenter(t *testing.T) {
	l := imagePack(t, logoScale, 100, 50) // 360x180 px at the top-left
	at := func(ms int) ui.Op {
		return imageOps(draw(l, ui.TitleState{Phase: ui.PhaseMenu, ElapsedMs: ms}), "images/logo.png")[0]
	}
	// Start: half scale around the center (180, 90): 180x90 at (90, 45).
	start := at(0)
	near(t, "start w", start.W, 180)
	near(t, "start x", start.X, 90)
	near(t, "start y", start.Y, 45)
	// Halfway, linear: scale 0.75, 270x135 at (45, 22.5).
	mid := at(200)
	near(t, "mid w", mid.W, 270)
	near(t, "mid x", mid.X, 45)
	// End and after: the layout size.
	end := at(400)
	near(t, "end w", end.W, 360)
	near(t, "end x", end.X, 0)
	near(t, "after w", at(5000).W, 360)
}

func TestTitle_LogoFadesIn(t *testing.T) {
	part := `"logo": { "type": "image", "image": "images/logo.png", "size": [50, null],
	"transition": { "in": { "kind": "fade", "ms": 400, "delayMs": 100, "easing": "linear" } } }`
	l := imagePack(t, part, 100, 50)
	alpha := func(ms int) float32 {
		return imageOps(draw(l, ui.TitleState{Phase: ui.PhaseMenu, ElapsedMs: ms}), "images/logo.png")[0].Color.A
	}
	near(t, "before the delay", alpha(50), 0)
	near(t, "halfway", alpha(300), 0.5)
	near(t, "done", alpha(500), 1)
}

func TestTitle_SpriteSheetFramesAndAnimationsPickTheirSourceRect(t *testing.T) {
	sheet := `{ "image": "cursor.png",
	  "frames": { "c0": [0, 0, 32, 32], "c1": [32, 0, 32, 32] },
	  "animations": { "blink": { "frames": [{ "frame": "c0", "ms": 300 }, { "frame": "c1", "ms": 300 }] } } }`
	extra := fstest.MapFS{"images/cursor.json": {Data: []byte(sheet)}, "images/cursor.png": {Data: pngOf(t, 64, 32)}}
	part := `"background": [{ "type": "image", "sheet": "images/cursor.json", "animation": "blink", "anchor": "topLeft" },
	                       { "type": "image", "sheet": "images/cursor.json", "frame": "c1", "anchor": "topLeft" }]`
	l := layoutOf(t, gameWith(t, part, extra))
	at := func(ms int) []ui.Op {
		return imageOps(draw(l, ui.TitleState{Phase: ui.PhaseMenu, ElapsedMs: ms}), "images/cursor.png")
	}
	first := at(0)
	near(t, "animated u0 at 0", first[0].U0, 0)
	near(t, "animated u1 at 0", first[0].U1, 0.5)
	near(t, "still frame u0", first[1].U0, 0.5)
	near(t, "animated u0 at 300", at(300)[0].U0, 0.5)
	near(t, "animated u0 after the loop", at(600)[0].U0, 0)
	// A 32 px frame at 3 px per pixel is 96 px.
	near(t, "frame w", first[0].W, 96)
}

func TestTitle_BuiltInLogoIsCenteredAtTheTopAfterItsTransition(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{})
	list := draw(ui.NewLayout(pack), ui.TitleState{Phase: ui.PhaseMenu, ElapsedMs: 10000})
	logo := imageOps(list, "images/logo.png")
	if len(logo) != 1 {
		t.Fatalf("logo ops = %+v", logo)
	}
	// size [100, null] = 720 px wide; 1269x176 image: 720*176/1269 = 99.858 high.
	// anchor top, offset [0, 14] = 100.8 px down: x = (1280-720)/2.
	near(t, "x", logo[0].X, 280)
	near(t, "y", logo[0].Y, 100.8)
	near(t, "w", logo[0].W, 720)
	near(t, "h", logo[0].H, 99.858)
	// The tiled grid over the background: 64 px cells at 3 px = 192 px, on 2880x720.
	grid := imageOps(list, "images/grid.png")
	near(t, "grid u1", grid[0].U1, 15)
	near(t, "grid v1", grid[0].V1, 3.75)
	near(t, "grid alpha", grid[0].Color.A, 0.6)
}

func TestDrawList_ResetKeepsItsCapacitySoAFrameAllocatesNothing(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{})
	l := ui.NewLayout(pack)
	var list ui.DrawList
	states := []ui.TitleState{
		{Phase: ui.PhasePressStart, ElapsedMs: 700, Version: "1.0"},
		menuState([]string{"Versus", "Options", "Exit"}, 1),
	}
	for _, s := range states {
		list.Reset()
		l.Title(&list, view720, s) // warm-up: faces and slices grow once
		allocs := testing.AllocsPerRun(50, func() {
			list.Reset()
			l.Title(&list, view720, s)
		})
		if allocs != 0 {
			t.Errorf("phase %v: %v allocations per frame, want 0", s.Phase, allocs)
		}
	}
}

func pivotMenu(pivot string) string {
	return `"menu": { "anchor": "center", "pivot": "` + pivot + `", "spacing": 6, "style": "left", "activeStyle": "left",
	"cursor": { "type": "rect", "size": [20, 6], "color": "#ff0000" } }`
}

func TestTitle_TheFirstEntryIsAtThePivotWhateverThePivot(t *testing.T) {
	// Rows are 6 units = 43.2 px; the anchor is the screen center (640, 360).
	// The cursor is as high as a row, so its y is the active row's top.
	cases := []struct {
		pivot string
		want  float32 // the first row's top
	}{
		{"top", 360},      // the row hangs from the point
		{"center", 338.4}, // the row is centered on the point: 360 - 21.6
		{"bottom", 316.8}, // the row ends on the point: 360 - 43.2
	}
	entries := []string{"a", "b", "c", "d", "e"}
	for _, c := range cases {
		l := layoutOf(t, gameWith(t, pivotMenu(c.pivot), nil))
		first := rectOps(draw(l, menuState(entries, 0)), red)
		near(t, c.pivot+" first row", first[0].Y, c.want)
		third := rectOps(draw(l, menuState(entries, 2)), red)
		near(t, c.pivot+" third row", third[0].Y, c.want+2*43.2)
	}
}
