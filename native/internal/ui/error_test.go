package ui_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/ui"
)

func errorLayout(t *testing.T, width string) *ui.Layout {
	t.Helper()
	parts := `"title": { "style": "center", "anchor": "top", "offset": [0, 4] },
	  "message": { "style": "left", "anchor": "topLeft", "pivot": "topLeft", "width": ` + width + ` },
	  "hint": { "style": "center", "anchor": "bottom", "offset": [0, -5] }`
	return layoutOf(t, screenGame(t, "error", parts, nil))
}

func drawError(l *ui.Layout, s ui.ErrorState) *ui.DrawList {
	var list ui.DrawList
	l.Error(&list, view720, s)
	return &list
}

func TestError_TheMessageIsWrappedInsideItsWidthAtTheStylesFontWordByWord(t *testing.T) {
	// 40 units = 288 px; size 5 = 36 px Noto Sans Regular. Measured widths:
	// "Cannot read the" 274.0 (+ " content" would be 413), "content folder." 249.6
	// (+ " Use" 322.8), "Use -content to" 258.8 (+ " point" 356.6), "point at it" 162.7.
	list := drawError(errorLayout(t, "40"), ui.ErrorState{Message: "Cannot read the content folder. Use -content to point at it"})
	got := onlyTexts(list)
	want := []string{"Cannot read the", "content folder.", "Use -content to", "point at it"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("lines = %q, want %q", got, want)
	}
	// Independently of the literals: every line fits, and its next word would not.
	m := newMeasurer(t, 36)
	for i, line := range got {
		if m.width(line) > 288 {
			t.Errorf("line %d is %.1f px, wider than 288", i, m.width(line))
		}
	}
	// Lines are 1.25 x 36 = 45 px apart from the box top (0), left aligned at x=0.
	met := m.face.Metrics()
	shift := (float32(met.Ascent) - float32(met.Descent)) / 128
	ops := textOpsInOrder(list)
	for i, op := range ops {
		near(t, "line x", op.X, 0)
		near(t, "line baseline", op.Y, 22.5+float32(i)*45+shift)
	}
}

func textOpsInOrder(l *ui.DrawList) []ui.Op {
	var out []ui.Op
	for _, op := range l.Ops {
		if op.Kind == ui.OpText {
			out = append(out, op)
		}
	}
	return out
}

func TestError_AWordLongerThanTheWidthIsBrokenWhereItStopsFitting(t *testing.T) {
	// "Extraordinarilylo" is 280.5 px, one letter more 302.8: the break falls after it.
	got := onlyTexts(drawError(errorLayout(t, "40"), ui.ErrorState{Message: "Extraordinarilylongword! ok"}))
	want := []string{"Extraordinarilylo", "ngword! ok"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestError_TitleAndHintArePartsAndAnEmptyMessageDrawsNoLines(t *testing.T) {
	list := drawError(errorLayout(t, "40"), ui.ErrorState{Title: "Oops", Hint: "Press Enter"})
	got := onlyTexts(list)
	if len(got) != 2 || got[0] != "Oops" || got[1] != "Press Enter" {
		t.Errorf("texts = %q, want the title and the hint", got)
	}
}

func TestError_APackWithoutAnErrorScreenBorrowsTheDefaultsWithItsBackground(t *testing.T) {
	l := layoutOf(t, gameWith(t, redBackground, nil)) // a pack with a title screen only
	var list ui.DrawList
	l.Error(&list, view720, ui.ErrorState{Title: "Something went wrong", Message: "x", Hint: "Press Enter to quit"})
	got := shownTexts(&list)
	if len(got) != 3 || got[0] != "SOMETHING WENT WRONG" || got[1] != "x" || got[2] != "Press Enter to quit" {
		t.Errorf("texts = %q, want the default error screen's title (uppercased), message and hint", got)
	}
	if len(imageOps(&list, "images/background.png")) != 0 {
		t.Errorf("the default screen's own assets come from the default pack, not the pack's")
	}
}

func TestError_AFrameAllocatesNothingOnceTheMessageIsWrapped(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	l := ui.NewLayout(pack)
	s := ui.ErrorState{Title: "Something went wrong", Hint: "Press Enter to quit",
		Message: "Cannot read the content: the chars folder is missing and the stages one too. Use -content to point at the content folder."}
	var list ui.DrawList
	l.Error(&list, view720, s)
	allocs := testing.AllocsPerRun(50, func() {
		list.Reset()
		l.Error(&list, view720, s)
	})
	if allocs != 0 {
		t.Errorf("%v allocations per frame, want 0", allocs)
	}
}

func TestOptions_ValueOffsetsPlaceEachValueColumnOfARow(t *testing.T) {
	menu := `"menu": { "anchor": "topLeft", "pivot": "topLeft", "spacing": 6, "style": "left", "activeStyle": "left",
	  "valueStyle": "value", "valueOffset": [40, 0], "valueOffsets": [[10, 0], [30, 2]] }`
	l := optionsLayout(t, menu)
	rows := []ui.Row{{Label: "A", Values: [2]string{"F", "a"}}, {Label: "Master", Values: [2]string{"90"}}}
	list := drawOptions(l, ui.OptionsState{Rows: rows, Column: -1})
	m := newMeasurer(t, 36)
	// Two values: one offset per column, right edges at 72 and 216 px; the
	// second column is also 2 units (14.4 px) lower.
	key, button, label := only(t, list, "F"), only(t, list, "a"), only(t, list, "A")
	near(t, "key x", key.X, 72-m.width("F"))
	near(t, "button x", button.X, 216-m.width("a"))
	near(t, "key y", key.Y, label.Y)
	near(t, "button y", button.Y, label.Y+14.4)
	// A row with one value uses valueOffset alone.
	near(t, "single value x", only(t, list, "90").X, 288-m.width("90"))
}

func TestOptions_AFlaggedRowUsesTheMenusFlaggedStyleElseTheRootWarningColor(t *testing.T) {
	rows := []ui.Row{{Label: "A", Values: [2]string{"F"}, Flagged: true}, {Label: "B", Values: [2]string{"G"}}}
	// With a flaggedStyle the row takes that style as is (color #12ab34, left aligned).
	styled := optionsLayout(t, `"menu": { "anchor": "topLeft", "pivot": "topLeft", "spacing": 6, "style": "left", "activeStyle": "left",
	  "flaggedStyle": "flag", "valueStyle": "value", "valueOffset": [40, 0] }`)
	list := drawOptions(styled, ui.OptionsState{Rows: rows, Cursor: 1, Column: -1})
	want := ui.Color{R: 0x12 / 255.0, G: 0xab / 255.0, B: 0x34 / 255.0, A: 1}
	for _, s := range []string{"A", "F"} {
		if got := only(t, list, s).Color; got != want {
			t.Errorf("%q color = %+v, want the flagged style's %+v", s, got, want)
		}
	}
	if only(t, list, "G").Color != orange {
		t.Errorf("an unflagged value lost its style")
	}
	// Without one, the row's own style tinted with the root's warningColor.
	root := strings.Replace(rootJSON, `"formatVersion": 1,`, `"formatVersion": 1, "warningColor": "#112233",`, 1)
	game := screenGame(t, "options", valuedMenu+`, "hint": { "style": "center", "anchor": "bottom" }`, fstest.MapFS{"screenpack.json": {Data: []byte(root)}})
	tinted := layoutOf(t, game)
	list = drawOptions(tinted, ui.OptionsState{Rows: rows, Cursor: 1, Column: -1, Notices: []string{"N"}})
	tint := ui.Color{R: 0x11 / 255.0, G: 0x22 / 255.0, B: 0x33 / 255.0, A: 1}
	for _, s := range []string{"A", "F", "N"} {
		if got := only(t, list, s).Color; got != tint {
			t.Errorf("%q color = %+v, want the root warningColor %+v", s, got, tint)
		}
	}
}

func TestError_ANewlineInTheMessageEndsTheLine(t *testing.T) {
	got := onlyTexts(drawError(errorLayout(t, "40"), ui.ErrorState{Message: "one\ntwo"}))
	want := []string{"one", "two"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %q, want %q", got, want)
	}
}
