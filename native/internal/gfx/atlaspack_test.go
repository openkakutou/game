package gfx

import "testing"

func TestShelfPacker_FillsARowThenStartsTheNextOneBelow(t *testing.T) {
	p := shelfPacker{size: 10}
	steps := []struct {
		w, h         int
		wantX, wantY int
	}{
		{4, 3, 0, 0}, // the first row
		{4, 5, 5, 0}, // 4 wide plus 1 px of padding; the row is now 5 high
		{4, 2, 0, 6}, // 5+1+4 > 10: the next row, below the 5-high one and its padding
	}
	for _, s := range steps {
		x, y, ok := p.place(s.w, s.h)
		if !ok || x != s.wantX || y != s.wantY {
			t.Errorf("place(%d,%d) = (%d,%d,%v), want (%d,%d,true)", s.w, s.h, x, y, ok, s.wantX, s.wantY)
		}
	}
}

func TestShelfPacker_ReportsAFullPageInsteadOfDroppingTheGlyph(t *testing.T) {
	p := shelfPacker{size: 10}
	if _, _, ok := p.place(10, 8); !ok {
		t.Fatal("a glyph that fills the page's width must fit an empty page")
	}
	if _, _, ok := p.place(10, 8); ok {
		t.Errorf("a second 8-high row does not fit in 10 rows: the packer must say so")
	}
}

func TestShelfPacker_RefusesAGlyphLargerThanThePage(t *testing.T) {
	p := shelfPacker{size: 10}
	if _, _, ok := p.place(11, 2); ok {
		t.Errorf("an 11 px wide glyph cannot fit a 10 px page")
	}
}
