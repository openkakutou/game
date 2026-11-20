package game

import (
	"os"
	"testing"
	"testing/fstest"
)

func charFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, data := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return fsys
}

func TestDiscoverCharacters_DisplayNameIsDisplaynameThenNameThenTheFolder(t *testing.T) {
	fsys := charFS(map[string]string{
		"chars/kfm/kfm.def":     "[Info]\nname = \"Kung Fu Man\"        ;internal\ndisplayname = \"Kung Fu Man DX\" ;shown\nauthor = \"Elecbyte\"\n[Files]\nsprite = kfm.sff\n",
		"chars/named/named.def": "[INFO]\nNAME = Named One\n",
		"chars/bare/bare.def":   "[Files]\nsprite = bare.sff\n",
		"chars/other/other.def": "[Files]\nname = not in info\n[Info]\nname = \"\"\n",
	})
	got, err := DiscoverCharacters(fsys, "chars")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Entry{
		"kfm":   {Name: "kfm", DefPath: "chars/kfm/kfm.def", Display: "Kung Fu Man DX", Author: "Elecbyte", Sprite: "chars/kfm/kfm.sff"},
		"named": {Name: "named", DefPath: "chars/named/named.def", Display: "Named One"},
		"bare":  {Name: "bare", DefPath: "chars/bare/bare.def", Display: "bare", Sprite: "chars/bare/bare.sff"},
		"other": {Name: "other", DefPath: "chars/other/other.def", Display: "other"},
	}
	if len(got) != len(want) {
		t.Fatalf("entries = %+v", got)
	}
	for _, e := range got {
		if e != want[e.Name] {
			t.Errorf("%s = %+v, want %+v", e.Name, e, want[e.Name])
		}
	}
}

func TestDiscoverStages_DisplayNameIsDisplaynameThenNameThenTheFileName(t *testing.T) {
	fsys := charFS(map[string]string{
		"stages/a.def":     "[Info]\nname = \"Alpha\"\ndisplayname = \"Alpha Arena\"\n[BGdef]\n",
		"stages/b.def":     "[Info]\nname = \"Beta\"\n[BGdef]\n",
		"stages/gamma.def": "[BGdef]\n",
	})
	got, err := DiscoverStages(fsys, "stages")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Alpha Arena", "Beta", "gamma"}
	for i, e := range got {
		if e.Display != want[i] {
			t.Errorf("stage %d display = %q, want %q", i, e.Display, want[i])
		}
	}
}

func TestSortEntries_OrdersByFolderDisplayNameOrAuthorIgnoringCase(t *testing.T) {
	mk := func() []Entry {
		return []Entry{
			{Name: "c", Display: "alpha", Author: "Zed"},
			{Name: "A", Display: "Charlie", Author: "amy"},
			{Name: "b", Display: "Bravo", Author: "Bob"},
		}
	}
	names := func(es []Entry) string {
		s := ""
		for _, e := range es {
			s += e.Name
		}
		return s
	}
	cases := []struct {
		by         string
		descending bool
		want       string
	}{
		{"folder", false, "Abc"},      // a, b, c
		{"displayName", false, "cbA"}, // alpha, Bravo, Charlie
		{"author", false, "Abc"},      // amy, Bob, Zed
		{"displayName", true, "Abc"},  // Charlie, Bravo, alpha
		{"folder", true, "cbA"},
	}
	for _, c := range cases {
		es := mk()
		SortEntries(es, c.by, c.descending)
		if got := names(es); got != c.want {
			t.Errorf("by %s descending=%v = %s, want %s", c.by, c.descending, got, c.want)
		}
	}
}

func TestGridMove_StaysInTheRowForLeftRightAndWrapsBetweenRowsForUpDown(t *testing.T) {
	// 10 cells in 4 columns: rows 0-3, 4-7 and 8-9 (a partial last row).
	cases := []struct {
		from int
		nav  Nav
		want int
	}{
		{0, NavRight, 1},
		{0, NavLeft, 3},  // wraps inside row 0
		{3, NavRight, 0}, // the same
		{9, NavRight, 8}, // the partial row wraps among its own two cells
		{8, NavLeft, 9},
		{1, NavDown, 5},
		{5, NavDown, 9},
		{9, NavDown, 1}, // past the last row: back to the top of the column
		{1, NavUp, 9},   // above the first row: the last row that has the column
		{3, NavUp, 7},   // row 2 has no column 3: skipped
		{7, NavDown, 3}, // likewise going down
		{6, NavConfirm, 6},
	}
	for _, c := range cases {
		if got := gridMove(c.from, c.nav, 4, 10); got != c.want {
			t.Errorf("from %d %v = %d, want %d", c.from, c.nav, got, c.want)
		}
	}
	// One column: a list.
	if got := gridMove(0, NavUp, 1, 3); got != 2 {
		t.Errorf("one column, up from 0 = %d, want 2", got)
	}
	// No columns set: one row of every cell.
	if got := gridMove(2, NavRight, 0, 3); got != 0 {
		t.Errorf("no columns, right from 2 = %d, want 0", got)
	}
	if got := gridMove(1, NavDown, 0, 3); got != 1 {
		t.Errorf("no columns, down = %d, want to stay on the only row", got)
	}
}

func TestFlow_SelectNavigatesTheGridIn2D(t *testing.T) {
	f := newTestFlow(t) // two characters
	f.GridColumns = 1
	f.Navigate(NavDown)
	if f.Cursor != 1 {
		t.Errorf("one column: down = %d, want 1", f.Cursor)
	}
	f.GridColumns = 2
	f.Navigate(NavDown) // one row only: nowhere to go
	if f.Cursor != 1 {
		t.Errorf("two columns, one row: down = %d, want to stay on 1", f.Cursor)
	}
	f.Navigate(NavRight)
	if f.Cursor != 0 {
		t.Errorf("right from the row's end = %d, want the row's start 0", f.Cursor)
	}
}

func TestFlow_StageSelectStepsWithLeftAndRightAndWraps(t *testing.T) {
	f := newTestFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm) // both picked: the stage screen
	if f.Screen != ScreenStage {
		t.Fatalf("screen = %v", f.Screen)
	}
	f.Navigate(NavLeft)
	if f.Cursor != len(f.Stages)-1 {
		t.Errorf("left from the first stage = %d, want the last", f.Cursor)
	}
	f.Navigate(NavRight)
	if f.Cursor != 0 {
		t.Errorf("right from the last stage = %d, want the first", f.Cursor)
	}
}

func TestLoadPortrait_ReadsSpriteGroup9000ImagesZeroAndOneOfTheCharactersSff(t *testing.T) {
	// Kung Fu Man's own sprite file (testdata/reference) holds the MUGEN
	// portraits: 9000,0 is 25x25 (select grid) and 9000,1 is 120x140 (large).
	fsys := os.DirFS(reference)
	cases := []struct {
		large bool
		w, h  int
	}{{false, 25, 25}, {true, 120, 140}}
	for _, c := range cases {
		px, w, h, err := LoadPortrait(fsys, "chars/kfm/kfm.sff", c.large)
		if err != nil || w != c.w || h != c.h || len(px) != w*h {
			t.Fatalf("large=%v: %dx%d (%d px) err=%v, want %dx%d", c.large, w, h, len(px), err, c.w, c.h)
		}
		opaque := 0
		for _, p := range px {
			if p.A != 0 {
				opaque++
			}
		}
		if opaque == 0 || opaque == len(px) {
			t.Errorf("large=%v: %d of %d pixels opaque, want a cut-out portrait", c.large, opaque, len(px))
		}
	}
}

func TestLoadPortrait_ACharacterWithoutPortraitSpritesIsAnErrorNotACrash(t *testing.T) {
	fsys := os.DirFS(content)
	if _, _, _, err := LoadPortrait(fsys, "chars/fighter/fighter.sff", true); err == nil {
		t.Errorf("the fixture fighter has no 9000 sprites: want an error")
	}
	if _, _, _, err := LoadPortrait(fsys, "chars/ghost/ghost.sff", false); err == nil {
		t.Errorf("a missing sprite file must be an error")
	}
	if _, _, _, err := LoadPortrait(fsys, "", false); err == nil {
		t.Errorf("a character without a sprite file must be an error")
	}
}

func TestFlow_SelectCursorsShowThePickerAtItsCursorAndAPickedSideAtItsPick(t *testing.T) {
	f := newTestFlow(t) // sides p1 then p2
	if got := f.SelectCursors(); got != [2]SideCursor{{Shown: true, Cursor: 0}, {}} {
		t.Errorf("before any pick = %+v, want only P1 shown on the cursor", got)
	}
	f.Navigate(NavRight) // P1 hovers the second character
	if got := f.SelectCursors(); got[0] != (SideCursor{Shown: true, Cursor: 1}) {
		t.Errorf("P1 hovering = %+v, want cursor 1", got[0])
	}
	f.Navigate(NavConfirm) // P1 picks 1; P2's cursor starts on its own pick (0)
	f.Navigate(NavLeft)
	got := f.SelectCursors()
	if got[0] != (SideCursor{Shown: true, Cursor: 1}) || got[1] != (SideCursor{Shown: true, Cursor: f.Cursor}) {
		t.Errorf("P2 picking = %+v, want P1 kept on its pick 1 and P2 on the cursor %d", got, f.Cursor)
	}
}
